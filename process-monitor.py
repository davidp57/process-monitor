"""
Process Monitor - Monitors a process and relaunches it if not running
Supports TUI display and output capture to logs or TUI window

Dependencies: See requirements.txt
Build executable: python build.py
"""


import psutil
import subprocess
import sys
import os
import time
import logging
import threading
import queue
import shutil
from pathlib import Path
from datetime import datetime
from typing import Optional, List, Dict
import click
import contextlib
import yaml
from rich.console import Console
from rich.live import Live
from rich.layout import Layout
from rich.panel import Panel
from rich.table import Table
from rich.text import Text
from rich.logging import RichHandler
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich import print as rprint

VERSION: str = "2.0.0"


def perform_file_copies(copy_operations: List[Dict[str, str]], logger: logging.Logger, stage: str = "") -> bool:
    """
    Perform file copy operations from a list of source/destination pairs.
    
    Args:
        copy_operations: List of dicts with 'source' and 'destination' keys
        logger: Logger instance for output
        stage: Description of when copies are happening (e.g., "pre-launch", "post-process")
    
    Returns:
        True if all copies succeeded, False if any failed
    """
    if not copy_operations:
        return True
    
    stage_prefix = f"[{stage}] " if stage else ""
    logger.info(f"{stage_prefix}Starting file copy operations ({len(copy_operations)} operations)")
    
    success_count = 0
    fail_count = 0
    
    for idx, operation in enumerate(copy_operations, 1):
        source = operation.get('source')
        destination = operation.get('destination')
        
        if not source or not destination:
            logger.error(f"{stage_prefix}Copy operation {idx}: Missing source or destination")
            fail_count += 1
            continue
        
        try:
            source_path = Path(source)
            dest_path = Path(destination)
            
            # Check if source exists
            if not source_path.exists():
                logger.error(f"{stage_prefix}Copy operation {idx}: Source not found: {source}")
                fail_count += 1
                continue
            
            # Create destination directory if it doesn't exist
            dest_path.parent.mkdir(parents=True, exist_ok=True)
            
            # Perform the copy
            if source_path.is_file():
                shutil.copy2(source_path, dest_path)
                logger.info(f"{stage_prefix}Copy operation {idx}: {source} -> {destination}")
                success_count += 1
            elif source_path.is_dir():
                if dest_path.exists():
                    shutil.rmtree(dest_path)
                shutil.copytree(source_path, dest_path)
                logger.info(f"{stage_prefix}Copy operation {idx}: {source} -> {destination} (directory)")
                success_count += 1
            else:
                logger.error(f"{stage_prefix}Copy operation {idx}: Source is neither file nor directory: {source}")
                fail_count += 1
                
        except PermissionError as e:
            logger.error(f"{stage_prefix}Copy operation {idx}: Permission denied - {e}")
            fail_count += 1
        except Exception as e:
            logger.error(f"{stage_prefix}Copy operation {idx}: Failed to copy {source} to {destination} - {e}")
            fail_count += 1
    
    logger.info(f"{stage_prefix}File copy operations completed: {success_count} succeeded, {fail_count} failed")
    return fail_count == 0


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
        pre_launch_copies: Optional[List[Dict[str, str]]] = None,
        post_process_copies: Optional[List[Dict[str, str]]] = None,
        wait_for_process: bool = False,
    ):
        self.process_name = process_name
        self.executable_path = Path(executable_path)
        self.check_interval = check_interval
        self.startup_delay = startup_delay
        self.capture_output = capture_output
        self.output_log = output_log
        self.working_dir = working_dir
        self.pre_launch_copies = pre_launch_copies or []
        self.post_process_copies = post_process_copies or []
        self.wait_for_process = wait_for_process

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
        """Check if process is running by name"""
        for proc in psutil.process_iter(['name', 'pid']):
            with contextlib.suppress(psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
                if str(proc.info['name']).lower() == self.process_name.lower():
                    return True
        return False
    
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
            # Perform pre-launch file copies
            if self.pre_launch_copies:
                self.logger.info("Performing pre-launch file copy operations")
                perform_file_copies(self.pre_launch_copies, self.logger, "pre-launch")
            
            self.logger.info(f"Launching {self.executable_path}")

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
                self.capture_output_from_process(creation_flags, env, log_handle)
            else:
                self.current_process = subprocess.Popen(
                    str(self.executable_path),
                    shell=True,
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

    def capture_output_from_process(self, creation_flags, env, log_handle):
        if sys.platform == 'win32':
            creation_flags |= subprocess.CREATE_NO_WINDOW

        self.current_process = subprocess.Popen(
            str(self.executable_path),
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
    
    def wait_for_process_to_start(self, timeout: Optional[float] = None) -> bool:
        """
        Wait for the process to appear in the process list.
        
        Args:
            timeout: Maximum time to wait in seconds. None means wait indefinitely.
        
        Returns:
            True if process was found, False if timeout occurred
        """
        self.logger.info(f"Waiting for process {self.process_name} to start...")
        self.process_status = "Waiting for process"
        
        start_time = time.time()
        while self.running:
            if self.is_process_running():
                self.logger.info(f"Process {self.process_name} detected!")
                return True
            
            if timeout and (time.time() - start_time) >= timeout:
                self.logger.warning(f"Timeout waiting for process {self.process_name}")
                return False
            
            time.sleep(self.check_interval)
        
        return False
    
    def monitor_loop(self):
        """Main monitoring loop"""
        self.running = True
        self.logger.info("Starting process monitor")
        
        # If wait_for_process is enabled, wait for process to appear first
        if self.wait_for_process:
            if not self.wait_for_process_to_start():
                self.logger.error("Failed to detect process, stopping monitor")
                return
        
        # Track if we've already done post-process copies
        post_copies_done = False
        
        while self.running:
            self.last_check_time = datetime.now()
            
            if not self.is_process_running():
                # If process was running and now stopped, do post-process copies
                if self.process_status == "Running" and self.post_process_copies and not post_copies_done:
                    self.logger.info("Process ended, performing post-process file copy operations")
                    perform_file_copies(self.post_process_copies, self.logger, "post-process")
                    post_copies_done = True
                
                self.logger.warning(f"{self.process_name} not found")
                self.process_status = "Not running - launching"
                self.launch_process()
                post_copies_done = False  # Reset for next cycle
            else:
                self.process_status = "Running"
            
            time.sleep(self.check_interval)
    
    def stop(self):
        """Stop monitoring"""
        self.running = False
        self.logger.info("Stopping process monitor")


class MonitorTUI:
    def __init__(self, monitor: ProcessMonitor):
        self.monitor = monitor
        self.console = Console()
        self.running = True
        
    def create_status_table(self) -> Table:
        """Create status information table"""
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
    
    def create_output_panel(self) -> Panel:
        """Create output display panel"""
        if not self.monitor.capture_output:
            return Panel(
                Text("Output capture disabled", style="dim italic"),
                title="Process Output",
                border_style="blue"
            )
        
        lines = self.monitor.get_output_lines(30)
        if not lines:
            content = Text("No output yet...", style="dim italic")
        else:
            content = Text("\n".join(lines))
        
        return Panel(
            content,
            title=f"Process Output (last {len(lines)} lines)",
            border_style="blue"
        )
    
    def create_layout(self) -> Layout:
        """Create the main layout"""
        layout = Layout()
        layout.split_column(
            Layout(name="header", size=3),
            Layout(name="status", size=10),
            Layout(name="output"),
            Layout(name="footer", size=3)
        )
        
        layout["header"].update(
            Panel(
                Text("Process Monitor", style="bold magenta", justify="center"),
                border_style="magenta"
            )
        )
        
        layout["status"].update(
            Panel(self.create_status_table(), title="Monitor Status", border_style="green")
        )
        
        layout["output"].update(self.create_output_panel())
        
        layout["footer"].update(
            Panel(
                Text("Press Ctrl+C to stop monitoring", style="dim", justify="center"),
                border_style="yellow"
            )
        )
        
        return layout
    
    def run(self):
        """Run the TUI"""
        # Start monitoring in background thread
        monitor_thread = threading.Thread(target=self.monitor.monitor_loop, daemon=True)
        monitor_thread.start()
        
        try:
            with Live(self.create_layout(), console=self.console, refresh_per_second=4) as live:
                while self.running:
                    live.update(self.create_layout())
                    time.sleep(0.25)
        except KeyboardInterrupt:
            self.console.print("\n[yellow]Stopping monitor...[/yellow]")
        finally:
            self.monitor.stop()


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
                      working_dir: Optional[str] = None, wait_for_process: bool = False,
                      no_tui: bool = False, verbose: bool = False) -> dict:
    """Get default configuration dictionary"""
    config = {
        'process_name': process_name,
        'executable': executable,
        'interval': interval,
        'startup_delay': startup_delay,
        'capture_output': capture_output,
        'wait_for_process': wait_for_process,
        'no_tui': no_tui,
        'verbose': verbose
    }
    # Only include optional fields if they have values
    if output_log:
        config['output_log'] = output_log
    if working_dir:
        config['working_dir'] = working_dir
    return config


@click.command()
@click.option('--process-name', '-p', help='Name of the process to monitor (e.g., notepad.exe)')
@click.option('--executable', '-e', type=click.Path(), help='Path to executable to launch')
@click.option('--interval', '-i', default=5.0, type=float, help='Check interval in seconds (default: 5.0)')
@click.option('--startup-delay', '-d', default=2.0, type=float, help='Delay after launching before resuming checks (default: 2.0)')
@click.option('--capture-output', '-c', is_flag=True, help='Capture stdout/stderr from the process')
@click.option('--output-log', '-o', type=click.Path(), help='Log file for captured output')
@click.option('--working-dir', '-w', type=click.Path(exists=True, file_okay=False, dir_okay=True), help='Working directory for the launched process')
@click.option('--wait-for-process', is_flag=True, help='Wait for process to appear before monitoring (do not launch)')
@click.option('--no-tui', is_flag=True, help='Run without TUI (headless mode)')
@click.option('--verbose', '-v', is_flag=True, help='Enable verbose logging')
@click.option('--config', type=click.Path(), help='YAML configuration file to load')
def main(process_name, executable, interval, startup_delay, capture_output, output_log, working_dir, wait_for_process, no_tui, verbose, config):
    """
    Process Monitor - Monitors a process and relaunches it if not running.
    
    Examples:
        python process_monitor.py -p notepad.exe -e "C:\\Windows\\System32\\notepad.exe" -c -o output.log
        python process_monitor.py --config my-config.yaml
    """
    # Load configuration from file if specified
    config_data = {}
    if config:
        config_data = load_config(config)
    
    # Merge CLI options with config file (CLI options take precedence)
    final_config = {
        'process_name': process_name or config_data.get('process_name'),
        'executable': executable or config_data.get('executable'),
        'interval': interval if interval != 5.0 else config_data.get('interval', 5.0),
        'startup_delay': startup_delay if startup_delay != 2.0 else config_data.get('startup_delay', 2.0),
        'capture_output': capture_output or config_data.get('capture_output', False),
        'output_log': output_log or config_data.get('output_log'),
        'working_dir': working_dir or config_data.get('working_dir'),
        'wait_for_process': wait_for_process or config_data.get('wait_for_process', False),
        'no_tui': no_tui or config_data.get('no_tui', False),
        'verbose': verbose or config_data.get('verbose', False)
    }
    
    # Validate required parameters
    if not final_config['process_name']:
        raise click.ClickException("Process name is required (--process-name or in config file)")
    
    # Executable is only required if not waiting for process
    if not final_config['wait_for_process']:
        if not final_config['executable']:
            raise click.ClickException("Executable path is required (--executable or in config file) unless --wait-for-process is used")
        
        # Validate executable exists (only if it's a simple path, not a command with arguments)
        executable_path = Path(final_config['executable'])
        if ' ' not in final_config['executable'] and not executable_path.exists():
            raise click.ClickException(f"Executable not found: {executable_path}")
    else:
        # When waiting for process, executable is optional (use empty string if not provided)
        if not final_config['executable']:
            final_config['executable'] = ''
    
    # Setup logging with timestamped filename
    timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
    process_monitor_log_file = f'process_monitor-{timestamp}.log'
    
    log_level = logging.DEBUG if final_config['verbose'] else logging.INFO
    logging.basicConfig(
        level=log_level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler() if final_config['no_tui'] else RichHandler(rich_tracebacks=True),
            logging.FileHandler(process_monitor_log_file)
        ]
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
    
    # Get file copy operations from config
    pre_launch_copies = config_data.get('pre_launch_copies', [])
    post_process_copies = config_data.get('post_process_copies', [])
    
    # Create monitor
    monitor = ProcessMonitor(
        process_name=final_config['process_name'],
        executable_path=final_config['executable'],
        check_interval=final_config['interval'],
        startup_delay=final_config['startup_delay'],
        capture_output=final_config['capture_output'],
        output_log=output_log_with_timestamp,
        working_dir=final_config['working_dir'],
        pre_launch_copies=pre_launch_copies,
        post_process_copies=post_process_copies,
        wait_for_process=final_config['wait_for_process']
    )
    
    if final_config['no_tui']:
        # Run in headless mode with rich output (similar to Python reference)
        console = Console()
        
        rprint(f"[bold blue]Process Monitor Started[/bold blue]")
        rprint(f"Monitoring process: [green]{final_config['process_name']}[/green]")
        if final_config['wait_for_process']:
            rprint(f"Mode: [yellow]Wait for process (no launch)[/yellow]")
        else:
            rprint(f"Launch path: [yellow]{final_config['executable']}[/yellow]")
        rprint(f"Check delay: [cyan]{final_config['interval']}[/cyan] seconds")
        rprint(f"Startup delay: [cyan]{final_config['startup_delay']}[/cyan] seconds")
        if final_config['output_log']:
            rprint(f"Log file: [magenta]{output_log_with_timestamp}[/magenta]")
        if final_config['working_dir']:
            rprint(f"Working directory: [magenta]{final_config['working_dir']}[/magenta]")
        rprint("-" * 50)
        
        # Check if process is already running or wait for it
        if final_config['wait_for_process']:
            if not monitor.is_process_running():
                rprint(f"[yellow]Waiting for process {final_config['process_name']} to start...[/yellow]")
            else:
                rprint(f"[green]Process {final_config['process_name']} is already running[/green]")
        else:
            if not monitor.is_process_running():
                rprint(f"[yellow]Process {final_config['process_name']} not found. Launching...[/yellow]")
                time.sleep(final_config['startup_delay'])
                monitor.launch_process()
                if monitor.current_process:
                    rprint(f"[green]Process launched successfully with PID: {monitor.current_process.pid}[/green]")
                else:
                    rprint(f"[red]Failed to launch process[/red]")
            else:
                rprint(f"[green]Process {final_config['process_name']} is already running[/green]")
        
        # Start monitoring loop with progress spinner
        try:
            with Progress(
                SpinnerColumn(),
                TextColumn("[progress.description]{task.description}"),
                console=console,
            ) as progress:
                task = progress.add_task("Monitoring...", total=None)
                monitor.running = True
                
                while monitor.running:
                    progress.update(task, description=f"Monitoring {final_config['process_name']}...")
                    monitor.last_check_time = datetime.now()
                    
                    if not monitor.is_process_running():
                        rprint(f"\n[yellow]Process {final_config['process_name']} not found. Launching...[/yellow]")
                        progress.update(task, description=f"Launching {final_config['process_name']}...")
                        monitor.process_status = "Not running - launching"
                        monitor.launch_process()
                        if monitor.current_process:
                            rprint(f"[green]Process launched successfully with PID: {monitor.current_process.pid}[/green]")
                            progress.update(task, description=f"Waiting for startup delay...")
                        else:
                            rprint(f"[red]Failed to launch process[/red]")
                    else:
                        monitor.process_status = "Running"
                        progress.update(task, description=f"Monitoring {final_config['process_name']}... [green]RUNNING[/green]")
                    
                    time.sleep(final_config['interval'])
        except KeyboardInterrupt:
            rprint("\n[bold red]Monitoring stopped by user.[/bold red]")
            monitor.stop()
    else:
        # Run with TUI
        tui = MonitorTUI(monitor)
        tui.run()


if __name__ == '__main__':
    main()