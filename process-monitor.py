"""
Process Monitor - Monitors a process and relaunches it if not running
Supports TUI display and output capture to logs or TUI window

Dependencies: See requirements.txt
Build executable: python build.py
"""
import signal
import threading
import sys
import time
import logging
import psutil
import subprocess
import os
import queue
import shlex
import ctypes
from pathlib import Path
from datetime import datetime
from typing import Optional
import typer
import contextlib
import yaml
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.text import Text
from rich.logging import RichHandler
from rich import print as rprint
from rich.progress import Progress, SpinnerColumn, TextColumn
# Textual TUI
from textual.app import App, ComposeResult
from textual.widgets import Header, Footer, Static, Log
from textual.containers import Vertical

class ProcessMonitor:
    def __init__(
        self,
        process_name: str,
        executable_path: str,
        check_interval: float = 5.0,
        startup_delay: float = 0,
        capture_output: bool = False,
        output_log: Optional[str] = None,
        working_dir: Optional[str] = None,
        kill_existing: bool = False,
        monitor_process: bool = True
    ):
        self.process_name = process_name
        self.executable_path = Path(executable_path)
        self.check_interval = check_interval
        self.startup_delay = startup_delay
        self.capture_output = capture_output
        self.output_log = output_log
        self.working_dir = working_dir
        self.kill_existing = kill_existing
        self.monitor_process = monitor_process

        # Status tracking
        self.running = False
        self.last_check_time = None
        self.process_status = "Not started"
        self.launch_count = 0
        self.current_process = None

        # Output capture
        self.output_queue = queue.Queue(maxsize=1000)
        self.output_lines = []
        self.max_output_lines = 500

        # Logging setup
        self.logger = logging.getLogger("ProcessMonitor")
        
    def is_process_running(self) -> bool:
        """Check if process is running by name and executable path"""
        for proc in psutil.process_iter(['name', 'pid', 'exe']):
            with contextlib.suppress(psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
                if str(proc.info['name']).lower() == self.process_name.lower():
                    return True
        return False
    
    def kill_existing_processes(self):
        """Kill existing processes with same name and executable path"""
        for proc in psutil.process_iter(['name', 'pid', 'exe']):
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
            self.logger.info(f"Launching {self.executable_path}")

            # Kill existing processes if requested
            if self.kill_existing:
                self.kill_existing_processes()

            # Prepare command
            cmd = shlex.split(str(self.executable_path)) if ' ' in str(self.executable_path) else [str(self.executable_path)]

            # Open log file if specified
            log_handle = None
            if self.output_log:
                log_handle = open(self.output_log, 'a', encoding='utf-8')
                self.logger.info(f"Logging output to: {self.output_log}")

            # Prepare environment variables - add ELECTRON_NO_ATTACH_CONSOLE for Electron apps
            env = os.environ.copy()
            env['ELECTRON_NO_ATTACH_CONSOLE'] = 'true'

            # Use CREATE_NO_WINDOW to prevent console window and output inheritance
            creation_flags = subprocess.CREATE_NEW_PROCESS_GROUP if sys.platform == 'win32' else 0
            # Always redirect stdout/stderr to pipes if we need to capture or log
            if self.capture_output or self.output_log:
                self.capture_output_from_process(creation_flags, env, log_handle, cmd)
            else:
                self.current_process = subprocess.Popen(
                    cmd,
                    shell=False,
                    cwd=self.working_dir,
                    env=env,
                    creationflags=creation_flags
                )

            self.launch_count += 1
            self.process_status = "Running"

            if self.startup_delay > 0:
                time.sleep(self.startup_delay)

        except Exception as e:
            self.logger.error(f"Failed to launch process: {e}")
            self.process_status = f"Launch failed: {e}"

    def capture_output_from_process(self, creation_flags, env, log_handle, cmd):
        if sys.platform == 'win32':
            creation_flags |= subprocess.CREATE_NO_WINDOW

        self.current_process = subprocess.Popen(
            cmd,
            shell=False,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            bufsize=-1,  # Use default buffering to avoid warnings
            cwd=self.working_dir,
            env=env,
            creationflags=creation_flags
        )

        # Start threads to read stdout and stderr
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

        # Store log handle for cleanup
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
        self.logger.info("Starting process monitor")
        
        while self.running:
            self.last_check_time = datetime.now()
            
            if not self.is_process_running():
                self.logger.warning(f"{self.process_name} not found")
                if self.monitor_process:
                    self.process_status = "Not running - launching"
                    self.launch_process()
                else:
                    self.process_status = "Not running - monitoring disabled"
            else:
                self.process_status = "Running"
            
            time.sleep(self.check_interval)
    
    def stop(self):
        """Stop monitoring"""
        self.running = False
        self.logger.info("Stopping process monitor")


class MonitorTextualApp(App):
    """Textual-based TUI for the ProcessMonitor."""
    BINDINGS = [("q", "quit", "Quit"),("k", "kill", "Kill and quit")]

    def __init__(self, monitor: ProcessMonitor):
        super().__init__()
        self.monitor = monitor
        self.status_static = Static()
        self.output_log = Log(highlight=False)
        self._last_output_len = 0

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Vertical(
            Static(Text("Process Monitor", style="bold magenta"), id="title"),
            self.status_static,
            Static(Text("Process Output", style="bold blue")),
            self.output_log,
            Footer(),
        )

    def on_mount(self) -> None:
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
        self.handle_exit()
        self.monitor.stop()

    def _create_status_table(self) -> Table:
        table = Table(show_header=False, box=None, padding=(0, 1))
        table.add_column("Key", style="cyan bold", width=20)
        table.add_column("Value", style="white")

        table.add_row("Process Name", self.monitor.process_name)
        table.add_row("Executable", str(self.monitor.executable_path))
        table.add_row("Check Interval", f"{self.monitor.check_interval}s")
        table.add_row("Startup Delay", f"{self.monitor.startup_delay}s")
        table.add_row("Launch Count", str(self.monitor.launch_count))

        status_color = "green" if self.monitor.process_status == "Running" else "yellow"
        table.add_row("Status", f"[{status_color}]{self.monitor.process_status}[/{status_color}]")

        if self.monitor.last_check_time:
            table.add_row("Last Check", self.monitor.last_check_time.strftime("%Y-%m-%d %H:%M:%S"))

        return table

    def refresh_ui(self) -> None:
        # Update status panel
        status_table = self._create_status_table()
        self.status_static.update(Panel(status_table, title="Monitor Status", border_style="green"))

        # Update output log incrementally if enabled
        if self.monitor.capture_output:
            lines = self.monitor.get_output_lines(500)
            current_len = len(lines)
            if current_len < self._last_output_len:
                # Buffer trimmed; reset
                self.output_log.clear()
                base = 0
            else:
                base = self._last_output_len
            for line in lines[base:]:
                self.output_log.write_line(line)
            self._last_output_len = current_len
        elif self._last_output_len == 0:
            self.output_log.clear()
            self.output_log.write_line("Output capture disabled")
            self._last_output_len = 1

    def action_kill(self) -> None:
        self.monitor.kill_existing_processes()
        self.exit()

    def handle_exit(self) -> None:
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


def get_default_config(process_name: Optional[str] = None, executable: Optional[str] = None,
                      interval: float = 5.0, startup_delay: float = 2.0,
                      capture_output: bool = False, output_log: Optional[str] = None,
                      working_dir: Optional[str] = None, tui: bool = True,
                      verbose: bool = False, kill_existing: bool = False, 
                      monitor_process: bool = True) -> dict:
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
    }
    # Only include optional fields if they have values
    if output_log:
        config['output_log'] = output_log
    if working_dir:
        config['working_dir'] = working_dir
    return config


app = typer.Typer()


@app.callback(invoke_without_command=True)
def main(
    config: Optional[str] = typer.Option(None, help='YAML configuration file to load'),
    process_name: str = typer.Option(None, help='Name of the process to monitor (e.g., notepad.exe)'),
    executable: str = typer.Option(None, help='Path to executable to launch'),
    interval: float = typer.Option(5.0, help='Check interval in seconds (default: 5.0)'),
    startup_delay: float = typer.Option(2.0, help='Delay after launching before resuming checks (default: 2.0)'),
    capture_output: bool = typer.Option(True, help='Capture stdout/stderr from the process'),
    output_log: Optional[str] = typer.Option(None, help='Log file for captured output'),
    working_dir: Optional[str] = typer.Option(None, help='Working directory for the launched process'),
    tui: Optional[bool] = typer.Option(True, '--tui/--no-tui', help='Use TUI mode'),
    verbose: bool = typer.Option(False, help='Enable verbose logging'),
    kill_existing: bool = typer.Option(False, help='Kill existing processes with same name and executable before launching and after exiting'),
    monitor: bool = typer.Option(True, '--monitor/--no-monitor', help='Relaunch missing process automatically (default: enabled)'),
):
    """
    Process Monitor - Monitors processes and performs file operations.

    Examples:
        python process_monitor.py -p notepad.exe -e "C:\\Windows\\System32\\notepad.exe" -c -o output.log
    """

    config_data = load_config(config) if config else {}
    # Merge CLI options with config file (CLI options take precedence)
    final_config = {
        'process_name': process_name or config_data.get('process_name'),
        'executable': executable or config_data.get('executable'),
        'interval': interval if interval != 5.0 else config_data.get('interval', 5.0),
        'startup_delay': startup_delay if startup_delay != 2.0 else config_data.get('startup_delay', 2.0),
        'capture_output': capture_output or config_data.get('capture_output', False),
        'output_log': output_log or config_data.get('output_log'),
        'working_dir': working_dir or config_data.get('working_dir'),
        'tui': tui if tui is not None else config_data.get('tui', True),
        'verbose': verbose or config_data.get('verbose', False),
        'kill_existing': kill_existing or config_data.get('kill_existing', False),
        'monitor_process': monitor if monitor is not None else config_data.get('monitor_process', True),
    }

    # Validate required parameters
    if not final_config['process_name']:
        typer.echo("Process name is required (--process-name or in config file)", err=True)
        raise typer.Exit(1)
    if not final_config['executable']:
        typer.echo("Executable path is required (--executable or in config file)", err=True)
        raise typer.Exit(1)

    # Validate interval and startup delay
    if final_config['interval'] <= 0:
        typer.echo("Check interval must be positive", err=True)
        raise typer.Exit(1)
    if final_config['startup_delay'] < 0:
        typer.echo("Startup delay must be non-negative", err=True)
        raise typer.Exit(1)

    # Validate executable exists (check the command's first part)
    cmd = shlex.split(final_config['executable'], posix=False)
    if not Path(cmd[0]).exists():
        typer.echo(f"Executable not found: {cmd[0]}", err=True)
        raise typer.Exit(1)

    # Setup logging with timestamped filename
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    process_monitor_log_file = f'process_monitor-{timestamp}.log'

    log_level = logging.DEBUG if final_config['verbose'] else logging.INFO
    handlers_list = [logging.FileHandler(process_monitor_log_file)]
    # Avoid attaching a console handler when using Textual TUI to prevent conflicts
    if not final_config['tui']:
        handlers_list.insert(0, RichHandler(rich_tracebacks=True))
    logging.basicConfig(
        level=log_level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=handlers_list
    )

    # Save current configuration as process-monitor-default-config.yaml
    save_config('process-monitor-default-config.yaml', get_default_config(**final_config))

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
    )

    # Run with TUI if requested
    if final_config['tui']:
        # Run with Textual TUI
        textual_app = MonitorTextualApp(monitor_obj)
        try:
            textual_app.run()
        finally:
            # Ensure cleanup even if Textual's on_shutdown isn't triggered
            monitor_obj.stop()
            textual_app.handle_exit()
    else:
        # Run in headless mode with rich output (similar to Python reference)
        console = Console()

        rprint("[bold blue]Process Monitor Started[/bold blue]")
        rprint(f"Monitoring process: [green]{final_config['process_name']}[/green]")
        rprint(f"Launch path: [yellow]{final_config['executable']}[/yellow]")
        rprint(f"Check delay: [cyan]{final_config['interval']}[/cyan] seconds")
        rprint(f"Startup delay: [cyan]{final_config['startup_delay']}[/cyan] seconds")
        if final_config['output_log']:
            rprint(f"Log file: [magenta]{output_log_with_timestamp}[/magenta]")
        if final_config['working_dir']:
            rprint(f"Working directory: [magenta]{final_config['working_dir']}[/magenta]")
        rprint("-" * 50)

        # Check if process is already running
        if monitor_obj.is_process_running():
            rprint(f"[green]Process {final_config['process_name']} is already running[/green]")

        elif final_config['monitor_process']:
            rprint(f"[yellow]Process {final_config['process_name']} not found. Launching...[/yellow]")
            time.sleep(final_config['startup_delay'])
            monitor_obj.launch_process()
            if monitor_obj.current_process:
                rprint(f"[green]Process launched successfully with PID: {monitor_obj.current_process.pid}[/green]")
            else:
                rprint("[red]Failed to launch process[/red]")
        else:
            rprint(f"[yellow]Process {final_config['process_name']} not found. Monitoring disabled; not launching.[/yellow]")
        # Start monitoring loop with progress spinner
        try:
            with Progress(
                                                    SpinnerColumn(),
                                                    TextColumn("[progress.description]{task.description}"),
                                                    console=console,
                                                ) as progress:
                task = progress.add_task("Monitoring...", total=None)
                monitor_obj.running = True

                while monitor_obj.running:
                    progress.update(task, description=f"Monitoring {final_config['process_name']}...")
                    monitor_obj.last_check_time = datetime.now()

                    if monitor_obj.is_process_running():
                        monitor_obj.process_status = "Running"
                        progress.update(task, description=f"Monitoring {final_config['process_name']}... [green]RUNNING[/green]")

                    elif final_config['monitor_process']:
                        rprint(f"\n[yellow]Process {final_config['process_name']} not found. Launching...[/yellow]")
                        progress.update(task, description=f"Launching {final_config['process_name']}...")
                        monitor_obj.process_status = "Not running - launching"
                        monitor_obj.launch_process()
                        if monitor_obj.current_process:
                            rprint(f"[green]Process launched successfully with PID: {monitor_obj.current_process.pid}[/green]")
                            progress.update(task, description="Waiting for startup delay...")
                        else:
                            rprint("[red]Failed to launch process[/red]")
                    else:
                        monitor_obj.process_status = "Not running - monitoring disabled"
                        progress.update(task, description=f"Monitoring {final_config['process_name']}... [yellow]NOT RUNNING (monitoring disabled)[/yellow]")
                    time.sleep(final_config['interval'])
        except KeyboardInterrupt:
            rprint("\n[bold red]Monitoring stopped by user.[/bold red]")
            monitor_obj.stop()


if __name__ == '__main__':
    app()