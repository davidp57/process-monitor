"""
Process Monitor - Monitors a process and relaunches it if not running
Supports TUI display and output capture to logs or TUI window

Dependencies: See requirements.txt
Build executable: python build.py
"""
import contextlib
import logging
import os
import queue
import shutil
import signal
import subprocess
import sys
import threading
import time
from datetime import datetime
from pathlib import Path
from typing import Optional

import psutil
import typer
import yaml
from rich import print as rprint
from rich.console import Console
from rich.logging import RichHandler
from rich.panel import Panel
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.table import Table
from rich.text import Text

# Textual TUI
from textual.app import App, ComposeResult
from textual.containers import Vertical
from textual.widgets import Footer, Header, Log, Static

# Constants
VERSION = "1.0.0"
MAX_OUTPUT_LINES = 500
MAX_LOG_LINES = 500
OUTPUT_QUEUE_SIZE = 1000
LOG_QUEUE_SIZE = 1000

class TUIHandler(logging.Handler):
    """Logging handler that queues log messages for TUI display."""

    def __init__(self, queue):
        super().__init__()
        self.log_queue = queue

    def emit(self, record):
        msg = self.format(record)
        with contextlib.suppress(queue.Full):
            self.log_queue.put_nowait(msg)

class ProcessMonitor:
    """Monitors a process and relaunches it if not running, with support for TUI and output capture."""

    def __init__(
        self,
        process_name: str,
        executable_path: str,
        check_interval: float = 0.5,
        startup_delay: float = 0,
        capture_output: bool = False,
        output_log: Optional[str] = None,
        working_dir: Optional[str] = None,
        kill_existing: bool = False,
        monitor_process: bool = True,
        lock_file: Optional[str] = None,
        initial_file_copies: Optional[list] = None,
        final_file_copies: Optional[list] = None
    ):
        """Initialize the ProcessMonitor with configuration parameters."""
        self.process_name = process_name
        self.executable_path = Path(executable_path)
        self.check_interval = check_interval
        self.startup_delay = startup_delay
        self.capture_output = capture_output
        self.output_log = output_log
        self.working_dir = working_dir
        self.kill_existing = kill_existing
        self.monitor_process = monitor_process
        self.lock_file = lock_file
        self.initial_file_copies = initial_file_copies or []
        self.final_file_copies = final_file_copies or []

        # Status tracking
        self.running = False
        self.last_check_time = None
        self.process_status = "Not started"
        self.launch_count = 0
        self.current_process = None

        # Output capture
        self.output_queue = queue.Queue(maxsize=OUTPUT_QUEUE_SIZE)
        self.output_lines = []
        self.max_output_lines = MAX_OUTPUT_LINES

        # Logging setup
        self.logger = logging.getLogger("ProcessMonitor")
        
    def is_process_running(self) -> bool:
        """Check if process is running by name"""
        for proc in psutil.process_iter(['name', 'pid']):
            with contextlib.suppress(psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
                if str(proc.info['name']).lower() == self.process_name.lower():
                    return True
        return False

    def perform_file_copies(self, copies_list):
        """Perform file copies from a list of source-destination pairs"""
        for copy in copies_list:
            source = copy['source']
            dest = copy['destination']
            try:
                shutil.copy(source, dest)
                self.logger.info(f"Copied {source} to {dest}")
            except Exception as e:
                self.logger.error(f"Failed to copy {source} to {dest}: {e}")

    def kill_existing_processes(self):
        """Kill existing processes with same name"""
        for proc in psutil.process_iter(['name', 'pid']):
            with contextlib.suppress(psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
                if str(proc.info['name']).lower() == self.process_name.lower():
                    proc.kill()
    
    def _output_reader_with_log(self, pipe, log_handle, prefix):
        """Read output from subprocess pipe and optionally write to log file"""
        try:
            for line in iter(pipe.readline, b''):
                if line:
                    decoded = line.decode('utf-8', errors='replace').rstrip()
                    timestamp = datetime.now().strftime("%H:%M:%S")
                    formatted = f"[{timestamp}] {prefix}: {decoded}"

                    # Add to queue for TUI
                    with contextlib.suppress(queue.Full):
                        self.output_queue.put_nowait(formatted)
                    
                    # Write to log file if handle provided
                    if log_handle:
                        log_handle.write(formatted + '\n')
                        log_handle.flush()
        except Exception as e:
            self.logger.error(f"Error reading output: {e}")
        finally:
            pipe.close()
    
    def launch_process(self):
        """Launch the executable and optionally capture output"""
        try:
            self._perform_initial_setup()
            log_handle = self._open_log_file() if self.output_log else None
            env = self._prepare_environment()
            creation_flags = self._get_creation_flags()
            self._launch_subprocess(creation_flags, env, log_handle)
            self._handle_post_launch()
        except Exception as e:
            self.logger.error(f"Failed to launch process: {e}")
            self.process_status = f"Launch failed: {e}"

    def _perform_initial_setup(self):
        """Perform initial file copies and kill existing processes if needed"""
        self.perform_file_copies(self.initial_file_copies)
        self.logger.info(f"Launching {self.executable_path}")
        if self.kill_existing:
            self.kill_existing_processes()

    def _launch_subprocess(self, creation_flags: int, env: dict, log_handle):
        """Launch the subprocess with or without output capture"""
        if self.capture_output or self.output_log:
            self._launch_with_capture(creation_flags, env, log_handle)
        else:
            self._launch_without_capture(creation_flags, env)

    def _handle_post_launch(self):
        """Handle actions after successful launch"""
        self.launch_count += 1
        self.process_status = "Running"
        if self.startup_delay > 0:
            time.sleep(self.startup_delay)

    def _open_log_file(self):
        """Open log file for output capture"""
        log_handle = open(self.output_log, 'a', encoding='utf-8')
        self.logger.info(f"Logging output to: {self.output_log}")
        return log_handle

    def _prepare_environment(self) -> dict:
        """Prepare environment variables for subprocess"""
        env = os.environ.copy()
        env['ELECTRON_NO_ATTACH_CONSOLE'] = 'true'
        return env

    def _get_creation_flags(self) -> int:
        """Get subprocess creation flags for the current platform"""
        return subprocess.CREATE_NEW_PROCESS_GROUP if sys.platform == 'win32' else 0

    def _launch_with_capture(self, creation_flags: int, env: dict, log_handle):
        """Launch process with output capture"""
        if sys.platform == 'win32':
            creation_flags |= subprocess.CREATE_NO_WINDOW
        cmd = str(self.executable_path)
        self.current_process = subprocess.Popen(
            cmd,
            shell=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=-1,
            cwd=self.working_dir,
            env=env,
            creationflags=creation_flags
        )
        self._start_output_threads(log_handle)

    def _launch_without_capture(self, creation_flags: int, env: dict):
        """Launch process without output capture"""
        cmd = str(self.executable_path)
        self.current_process = subprocess.Popen(
            cmd,
            shell=False,
            cwd=self.working_dir,
            env=env,
            creationflags=creation_flags
        )

    def _start_output_threads(self, log_handle):
        """Start threads for reading stdout and stderr"""
        stdout_thread = threading.Thread(
            target=self._output_reader_with_log,
            args=(self.current_process.stdout, log_handle, "STDOUT"),
            daemon=True
        )
        stderr_thread = threading.Thread(
            target=self._output_reader_with_log,
            args=(self.current_process.stderr, log_handle, "STDERR"),
            daemon=True
        )
        stdout_thread.start()
        stderr_thread.start()
        if log_handle:
            self.current_process._log_handle = log_handle

    
    def get_output_lines(self, max_lines: int = 100):
        """Get recent output lines for display"""
        # Drain queue into list
        while not self.output_queue.empty():
            try:
                line = self.output_queue.get_nowait()
                self.output_lines.append(line)
                if len(self.output_lines) > self.max_output_lines:
                    self.output_lines.pop(0)
            except queue.Empty:
                break

        return self.output_lines[-max_lines:]
    
    def monitor_loop(self):
        """Main monitoring loop"""
        self.running = True
        self.first_launch_attempted = False
        self.logger.info("Starting process monitor")

        while self.running:
            self.last_check_time = datetime.now()
            self._check_and_handle_process_status()
            time.sleep(self.check_interval)

    def _check_and_handle_process_status(self):
        """Check process status and handle launching or stopping"""
        if not self.is_process_running():
            self.logger.warning(f"{self.process_name} not found")
            if self.monitor_process or not self.first_launch_attempted:
                self._launch_process_if_needed()
            else:
                self._disable_monitoring()
        else:
            self.process_status = "Running"

    def _launch_process_if_needed(self):
        """Launch the process and wait for lock file"""
        self.process_status = "Not running - launching"
        self.launch_process()
        self.wait_for_lock_file()
        self.first_launch_attempted = True

    def _disable_monitoring(self):
        """Disable monitoring when process is not running and monitoring is off"""
        self.process_status = "Not running - monitoring disabled"
        self.running = False
        self.stop()
    
    def wait_for_lock_file(self):
        """Wait for the lock file to be created if specified"""
        if self.lock_file:
            self.logger.info(f"Waiting for lock file: {self.lock_file}")
            while not Path(self.lock_file).exists():
                time.sleep(0.1)
            self.logger.info(f"Lock file found: {self.lock_file}")

    def stop(self):
        """Stop monitoring"""
        self.perform_file_copies(self.final_file_copies)
        self.running = False
        self.logger.info("Stopping process monitor")


class MonitorTextualApp(App):
    """Textual-based TUI for the ProcessMonitor."""
    TITLE = f"Process Monitor {VERSION}"
    BINDINGS = [("q", "quit", "Quit"),("k", "kill", "Kill and quit")]

    def __init__(self, monitor: ProcessMonitor, log_queue: queue.Queue, exit_delay: float, wait_keypress: bool):
        """Initialize the Textual TUI app with monitor and configuration."""
        super().__init__()
        self.monitor = monitor
        self.log_queue = log_queue
        self.exit_delay = exit_delay
        self.wait_keypress = wait_keypress
        self.log_lines = []
        self.max_log_lines = MAX_LOG_LINES
        self._last_log_len = 0
        self.status_static = Static()
        self.output_log = Log(highlight=False)
        self.script_log = Log(highlight=False)
        self.exit_static = Static()
        self._last_output_len = 0
        self.exiting = False

    def compose(self) -> ComposeResult:
        """Compose the TUI layout with header, status, logs, and footer."""
        yield Header(show_clock=True)
        yield Vertical(
            Static(Text("Process Monitor", style="bold magenta"), id="title"),
            self.status_static,
            Static(Text("Process Output", style="bold blue")),
            self.output_log,
            Static(Text("Script Logs", style="bold green")),
            self.script_log,
            self.exit_static,
            Footer(),
        )

    def on_mount(self) -> None:
        """Set up signal handlers, start monitor thread, and schedule UI refresh."""
        # Register signal handlers after app starts
        signal.signal(signal.SIGINT, self.handle_signal)
        signal.signal(signal.SIGTERM, self.handle_signal)
        monitor_thread = threading.Thread(target=self.monitor.monitor_loop, daemon=True)
        monitor_thread.start()
        # Periodically refresh UI
        self.set_interval(0.25, self.refresh_ui)
    
    def handle_signal(self, signum, frame):
        """Handle Ctrl+C and kill signals"""
        self.exit() 

    def on_shutdown(self) -> None:
        """Handle app shutdown by cleaning up monitor and exit logic."""
        self.handle_exit()
        self.monitor.stop()

    def _create_status_table(self) -> Table:
        """Create a status table displaying monitor information."""
        table = Table(show_header=False, box=None, padding=(0, 1))
        table.add_column("Key", style="cyan bold", width=20)
        table.add_column("Value", style="white")

        table.add_row("Process Name", self.monitor.process_name)
        table.add_row("Executable", str(self.monitor.executable_path))
        table.add_row("Check Interval", f"{self.monitor.check_interval}s")
        table.add_row("Startup Delay", f"{self.monitor.startup_delay}s")
        table.add_row("Launch Count", str(self.monitor.launch_count))
        table.add_row("Lock File", self.monitor.lock_file or "None")

        status_color = "green" if self.monitor.process_status == "Running" else "yellow"
        table.add_row("Status", f"[{status_color}]{self.monitor.process_status}[/{status_color}]")

        if self.monitor.last_check_time:
            table.add_row("Last Check", self.monitor.last_check_time.strftime("%Y-%m-%d %H:%M:%S"))

        return table

    def get_log_lines(self, max_lines: int = 100):
        """Get recent log lines for display"""
        while not self.log_queue.empty():
            try:
                line = self.log_queue.get_nowait()
                self.log_lines.append(line)
                if len(self.log_lines) > self.max_log_lines:
                    self.log_lines.pop(0)
            except queue.Empty:
                break

        return self.log_lines[-max_lines:]

    def refresh_ui(self) -> None:
        """Refresh all UI components with current monitor state."""
        self._update_status_panel()
        self._update_output_log()
        self._update_script_log()
        self._handle_exit_if_needed()

    def _update_status_panel(self):
        """Update the status panel with current monitor information"""
        status_table = self._create_status_table()
        self.status_static.update(Panel(status_table, title="Monitor Status", border_style="green"))

    def _update_output_log(self):
        """Update the process output log incrementally"""
        if self.monitor.capture_output:
            self._update_output_lines()
        elif self._last_output_len == 0:
            self.output_log.clear()
            self.output_log.write_line("Output capture disabled")
            self._last_output_len = 1

    def _update_output_lines(self):
        """Update output lines incrementally"""
        lines = self.monitor.get_output_lines(500)
        current_len = len(lines)
        if current_len < self._last_output_len:
            self.output_log.clear()
            base = 0
        else:
            base = self._last_output_len
        for line in lines[base:]:
            self.output_log.write_line(line)
        self._last_output_len = current_len

    def _update_script_log(self):
        """Update the script log incrementally"""
        log_lines = self.get_log_lines(500)
        current_log_len = len(log_lines)
        if current_log_len < self._last_log_len:
            self.script_log.clear()
            base = 0
        else:
            base = self._last_log_len
        for line in log_lines[base:]:
            self.script_log.write_line(line)
        self._last_log_len = current_log_len

    def _handle_exit_if_needed(self):
        """Handle exit logic when monitor has stopped"""
        if not self.monitor.running and not self.exiting:
            self.exiting = True
            if self.wait_keypress:
                self.exit_static.update("Press 'q' to exit")
            elif self.exit_delay > 0:
                self.exit_static.update(f"Exiting in {self.exit_delay} seconds")
                self.set_timer(self.exit_delay, self.exit)
            else:
                self.exit()

    def action_quit(self) -> None:
        """Handle quit action: exit if already exiting, otherwise kill processes and exit."""
        if self.exiting:
            self.exit()
        else:
            self.monitor.kill_existing_processes()
            self.exit()

    def action_kill(self) -> None:
        """Kill existing processes and exit the application."""
        self.monitor.kill_existing_processes()
        self.exit()

    def handle_exit(self) -> None:
        """Kill existing processes if configured to do so on exit."""
        if self.monitor.kill_existing:
            self.monitor.kill_existing_processes()

def load_config(config_file: str) -> dict:
    """Load configuration from YAML file"""
    try:
        with open(config_file, 'r', encoding='utf-8') as f:
            return yaml.safe_load(f) or {}
    except FileNotFoundError:
        logging.warning(f"Config file {config_file} not found, using defaults")
        return {}
    except yaml.YAMLError as e:
        logging.error(f"Error parsing config file {config_file}: {e}")
        return {}
    except Exception as e:
        logging.error(f"Error loading config file {config_file}: {e}")
        return {}


def save_config(config_file: str, config: dict):
    """Save configuration to YAML file"""
    try:
        with open(config_file, 'w', encoding='utf-8') as f:
            yaml.safe_dump(config, f, default_flow_style=False, indent=2, sort_keys=True)
        logging.info(f"Configuration saved to {config_file}")
    except Exception as e:
        logging.error(f"Error saving config file {config_file}: {e}")


def load_and_merge_config(config_file: Optional[str], cli_args: dict) -> dict:
    """Load config from file and merge with CLI args, CLI takes precedence"""
    config_data = load_config(config_file) if config_file else {}
    return {
        'process_name': cli_args.get('process_name') or config_data.get('process_name'),
        'executable': cli_args.get('executable') or config_data.get('executable'),
        'interval': cli_args.get('interval', config_data.get('interval', 0.5)),
        'startup_delay': cli_args.get('startup_delay', config_data.get('startup_delay', 2.0)),
        'capture_output': cli_args.get('capture_output', config_data.get('capture_output', False)),
        'output_log': cli_args.get('output_log') or config_data.get('output_log'),
        'working_dir': cli_args.get('working_dir') or config_data.get('working_dir'),
        'tui': cli_args.get('tui', config_data.get('tui', True)),
        'verbose': cli_args.get('verbose', config_data.get('verbose', False)),
        'kill_existing': cli_args.get('kill_existing', config_data.get('kill_existing', False)),
        'monitor_process': cli_args.get('monitor', config_data.get('monitor_process', True)),
        'lock_file': cli_args.get('lock_file') or config_data.get('lock_file'),
        'initial_file_copies': config_data.get('initial_file_copies', []),
        'final_file_copies': config_data.get('final_file_copies', []),
        'exit_delay': cli_args.get('exit_delay', config_data.get('exit_delay', 0.0)),
        'wait_keypress': cli_args.get('wait_keypress', config_data.get('wait_keypress', False)),
    }


def validate_config(config: dict):
    """Validate configuration parameters"""
    if not config['process_name']:
        typer.echo("Process name is required (--process-name or in config file)", err=True)
        raise typer.Exit(1)
    if not config['executable']:
        typer.echo("Executable path is required (--executable or in config file)", err=True)
        raise typer.Exit(1)
    if config['interval'] <= 0:
        typer.echo("Check interval must be positive", err=True)
        raise typer.Exit(1)
    if config['startup_delay'] < 0:
        typer.echo("Startup delay must be non-negative", err=True)
        raise typer.Exit(1)


def setup_logging(verbose: bool, use_tui: bool) -> str:
    """Setup logging configuration and return log file path"""
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    process_monitor_log_file = f'process_monitor-{timestamp}.log'

    log_level = logging.DEBUG if verbose else logging.INFO
    handlers_list = [logging.FileHandler(process_monitor_log_file)]
    # Avoid attaching a console handler when using Textual TUI to prevent conflicts
    if not use_tui:
        handlers_list.insert(0, RichHandler(rich_tracebacks=True))
    logging.basicConfig(
        level=log_level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=handlers_list
    )
    return process_monitor_log_file


def get_default_config(process_name: Optional[str] = None, executable: Optional[str] = None,
                       interval: float = 0.5, startup_delay: float = 2.0,
                       capture_output: bool = False, output_log: Optional[str] = None,
                       working_dir: Optional[str] = None, tui: bool = True,
                       verbose: bool = False, kill_existing: bool = False,
                       monitor_process: bool = True, lock_file: Optional[str] = None,
                       initial_file_copies: Optional[list] = None, final_file_copies: Optional[list] = None,
                       exit_delay: float = 0.0, wait_keypress: bool = False) -> dict:
    """Get default configuration dictionary"""
    config = {
        'process_name': process_name,
        'executable': executable,
        'interval': interval,
        'startup_delay': startup_delay,
        'capture_output': capture_output,
        'tui': tui,
        'verbose': verbose,
        'kill_existing': kill_existing,
        'monitor_process': monitor_process,
        'exit_delay': exit_delay,
        'wait_keypress': wait_keypress,
    }
    # Only include optional fields if they have values
    if output_log:
        config['output_log'] = output_log
    if working_dir:
        config['working_dir'] = working_dir
    if lock_file:
        config['lock_file'] = lock_file
    if initial_file_copies:
        config['initial_file_copies'] = initial_file_copies
    if final_file_copies:
        config['final_file_copies'] = final_file_copies
    return config


def run_tui_mode(monitor_obj: ProcessMonitor, config: dict):
    """Run the process monitor with TUI interface"""
    # Set up log queue and handler for TUI
    log_queue = queue.Queue(maxsize=LOG_QUEUE_SIZE)
    tui_handler = TUIHandler(log_queue)
    logging.getLogger("ProcessMonitor").addHandler(tui_handler)
    # Run with Textual TUI
    textual_app = MonitorTextualApp(monitor_obj, log_queue, config['exit_delay'], config['wait_keypress'])
    try:
        textual_app.run()
    finally:
        # Ensure cleanup even if Textual's on_shutdown isn't triggered
        monitor_obj.stop()
        textual_app.handle_exit()


def _handle_initial_process_check(monitor_obj: ProcessMonitor, config: dict):
    """Handle initial check and launch of the process"""
    if monitor_obj.is_process_running():
        rprint(f"[green]Process {config['process_name']} is already running[/green]")
    else:
        rprint(f"[yellow]Process {config['process_name']} not found. Launching...[/yellow]")
        time.sleep(config['startup_delay'])
        monitor_obj.launch_process()
        monitor_obj.wait_for_lock_file()
        if monitor_obj.current_process:
            rprint(f"[green]Process launched successfully with PID: {monitor_obj.current_process.pid}[/green]")
        else:
            rprint("[red]Failed to launch process[/red]")


def _run_monitoring_loop(monitor_obj: ProcessMonitor, config: dict, console: Console):
    """Run the main monitoring loop with progress display"""
    try:
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
        ) as progress:
            task = progress.add_task("Monitoring...", total=None)
            monitor_obj.running = True

            while monitor_obj.running:
                progress.update(task, description=f"Monitoring {config['process_name']}...")
                monitor_obj.last_check_time = datetime.now()

                if monitor_obj.is_process_running():
                    monitor_obj.process_status = "Running"
                    progress.update(task, description=f"Monitoring {config['process_name']}... [green]RUNNING[/green]")
                elif config['monitor_process']:
                    rprint(f"\n[yellow]Process {config['process_name']} not found. Launching...[/yellow]")
                    progress.update(task, description=f"Launching {config['process_name']}...")
                    monitor_obj.process_status = "Not running - launching"
                    monitor_obj.launch_process()
                    if monitor_obj.current_process:
                        rprint(f"[green]Process launched successfully with PID: {monitor_obj.current_process.pid}[/green]")
                        progress.update(task, description="Waiting for startup delay...")
                    else:
                        rprint("[red]Failed to launch process[/red]")
                else:
                    monitor_obj.process_status = "Not running - monitoring disabled"
                    progress.update(task, description=f"Monitoring {config['process_name']}... [yellow]NOT RUNNING (monitoring disabled)[/yellow]")
                    monitor_obj.running = False
                time.sleep(config['interval'])
        # Stop monitoring after loop exits naturally
        monitor_obj.stop()
    except KeyboardInterrupt:
        rprint("\n[bold red]Monitoring stopped by user.[/bold red]")
        monitor_obj.stop()


def run_headless_mode(monitor_obj: ProcessMonitor, config: dict, output_log_with_timestamp: Optional[str]):
    """Run the process monitor in headless mode with rich output"""
    console = Console()

    rprint("[bold blue]Process Monitor Started[/bold blue]")
    rprint(f"Monitoring process: [green]{config['process_name']}[/green]")
    rprint(f"Launch path: [yellow]{config['executable']}[/yellow]")
    rprint(f"Check delay: [cyan]{config['interval']}[/cyan] seconds")
    rprint(f"Startup delay: [cyan]{config['startup_delay']}[/cyan] seconds")
    if config['output_log']:
        rprint(f"Log file: [magenta]{output_log_with_timestamp}[/magenta]")
    if config['working_dir']:
        rprint(f"Working directory: [magenta]{config['working_dir']}[/magenta]")
    rprint("-" * 50)

    # Initial process check and launch
    _handle_initial_process_check(monitor_obj, config)

    # Start monitoring loop with progress spinner
    _run_monitoring_loop(monitor_obj, config, console)

    # Apply exit delay and keypress wait
    if config['exit_delay'] > 0:
        rprint(f"[yellow]Waiting {config['exit_delay']} seconds before exiting...[/yellow]")
        time.sleep(config['exit_delay'])
    if config['wait_keypress']:
        input("Press any key to exit...")


app = typer.Typer()


@app.callback(invoke_without_command=True)
def main(
    config: Optional[str] = typer.Option(None, help='YAML configuration file to load and save final config to'),
    process_name: str = typer.Option(None, help='Name of the process to monitor (e.g., notepad.exe)'),
    executable: str = typer.Option(None, help='Path to executable to launch'),
    interval: float = typer.Option(0.5, help='Check interval in seconds (default: 0.5)'),
    startup_delay: float = typer.Option(2.0, help='Delay after launching before resuming checks (default: 2.0)'),
    capture_output: bool = typer.Option(True, help='Capture stdout/stderr from the process'),
    output_log: Optional[str] = typer.Option(None, help='Log file for captured output'),
    working_dir: Optional[str] = typer.Option(None, help='Working directory for the launched process'),
    tui: Optional[bool] = typer.Option(True, '--tui/--no-tui', help='Use TUI mode'),
    verbose: bool = typer.Option(False, help='Enable verbose logging'),
    kill_existing: bool = typer.Option(False, help='Kill existing processes with same name and executable before launching and after exiting'),
    monitor: bool = typer.Option(None, '--monitor/--no-monitor', help='Relaunch missing process automatically (default: enabled)'),
    lock_file: Optional[str] = typer.Option(None, help='Path to lock file to wait for after launching the process'),
    exit_delay: float = typer.Option(0.0, help='Delay in seconds before exiting the app'),
    wait_keypress: bool = typer.Option(False, help='Wait for keypress before exiting'),
):
    """
    Process Monitor - Monitors processes and performs file operations.

    Examples:
        python process_monitor.py -p notepad.exe -e "C:\\Windows\\System32\\notepad.exe" -c -o output.log
    """

    cli_args = {
        'process_name': process_name,
        'executable': executable,
        'interval': interval,
        'startup_delay': startup_delay,
        'capture_output': capture_output,
        'output_log': output_log,
        'working_dir': working_dir,
        'tui': tui,
        'verbose': verbose,
        'kill_existing': kill_existing,
        'monitor': monitor,
        'lock_file': lock_file,
        'exit_delay': exit_delay,
        'wait_keypress': wait_keypress,
    }
    final_config = load_and_merge_config(config, cli_args)
    validate_config(final_config)

    # Setup logging
    process_monitor_log_file = setup_logging(final_config['verbose'], final_config['tui'])

    # Add timestamp to output log filename if specified
    output_log_with_timestamp = None
    if final_config['output_log']:
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        # Split the filename and extension
        path_obj = Path(final_config['output_log'])
        stem = path_obj.stem
        suffix = path_obj.suffix
        if path_obj.parent and str(path_obj.parent) != '.':
            output_log_with_timestamp = str(path_obj.parent / f"{stem}-{timestamp}{suffix}")
        else:
            output_log_with_timestamp = f"{stem}-{timestamp}{suffix}"

    # Create monitor
    monitor_obj = ProcessMonitor(
        process_name=final_config['process_name'],
        executable_path=final_config['executable'],
        check_interval=final_config['interval'],
        startup_delay=final_config['startup_delay'],
        capture_output=final_config['capture_output'],
        output_log=output_log_with_timestamp,
        working_dir=final_config['working_dir'],
        kill_existing=final_config['kill_existing'],
        monitor_process=final_config['monitor_process'],
        lock_file=final_config['lock_file'],
        initial_file_copies=final_config['initial_file_copies'],
        final_file_copies=final_config['final_file_copies'],
    )

    # Save configuration if config file was specified
    if config:
        save_config(config, get_default_config(**final_config))

    # Run with TUI if requested
    if final_config['tui']:
        run_tui_mode(monitor_obj, final_config)
    else:
        run_headless_mode(monitor_obj, final_config, output_log_with_timestamp)


if __name__ == '__main__':
    app()