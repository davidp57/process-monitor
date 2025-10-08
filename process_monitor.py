"""
Process Monitor - Monitors a process and relaunches it if not running
Supports TUI display and output capture to logs or TUI window

Dependencies: See requirements.txt
Build executable: python build.py
"""

import psutil
import subprocess
import time
import logging
import threading
import queue
from pathlib import Path
from datetime import datetime
from typing import Optional
import click
import yaml
from rich.console import Console
from rich.live import Live
from rich.layout import Layout
from rich.panel import Panel
from rich.table import Table
from rich.text import Text
from rich.logging import RichHandler

VERSION: str = "1.0.0"

class ProcessMonitor:
    def __init__(
        self,
        process_name: str,
        executable_path: str,
        check_interval: float = 5.0,
        startup_delay: float = 0,
        capture_output: bool = False,
        output_log: Optional[str] = None,
    ):
        self.process_name = process_name
        self.executable_path = Path(executable_path)
        self.check_interval = check_interval
        self.startup_delay = startup_delay
        self.capture_output = capture_output
        self.output_log = output_log
        
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
            try:
                if proc.info['name'] == self.process_name:
                    return True
            except (psutil.NoSuchProcess, psutil.AccessDenied, psutil.ZombieProcess):
                pass
        return False
    
    def _output_reader(self, pipe, prefix):
        """Read output from subprocess pipe"""
        try:
            for line in iter(pipe.readline, b''):
                if line:
                    decoded = line.decode('utf-8', errors='replace').rstrip()
                    timestamp = datetime.now().strftime("%H:%M:%S")
                    formatted = f"[{timestamp}] {prefix}: {decoded}"
                    
                    # Add to queue for TUI
                    try:
                        self.output_queue.put_nowait(formatted)
                    except queue.Full:
                        pass
                    
                    # Log to file if specified
                    if self.output_log:
                        with open(self.output_log, 'a', encoding='utf-8') as f:
                            f.write(formatted + '\n')
        except Exception as e:
            self.logger.error(f"Error reading output: {e}")
        finally:
            pipe.close()
    
    def launch_process(self):
        """Launch the executable and optionally capture output"""
        try:
            self.logger.info(f"Launching {self.executable_path}")
            
            if self.capture_output:
                self.current_process = subprocess.Popen(
                    [str(self.executable_path)],
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    bufsize=1
                )
                
                # Start threads to read stdout and stderr
                stdout_thread = threading.Thread(
                    target=self._output_reader,
                    args=(self.current_process.stdout, "STDOUT"),
                    daemon=True
                )
                stderr_thread = threading.Thread(
                    target=self._output_reader,
                    args=(self.current_process.stderr, "STDERR"),
                    daemon=True
                )
                stdout_thread.start()
                stderr_thread.start()
            else:
                self.current_process = subprocess.Popen([str(self.executable_path)])
            
            self.launch_count += 1
            self.process_status = "Running"
            
            if self.startup_delay > 0:
                time.sleep(self.startup_delay)
                
        except Exception as e:
            self.logger.error(f"Failed to launch process: {e}")
            self.process_status = f"Launch failed: {e}"
    
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
                self.process_status = "Not running - launching"
                self.launch_process()
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
                      no_tui: bool = False, verbose: bool = False) -> dict:
    """Get default configuration dictionary"""
    return {
        'process_name': process_name,
        'executable': executable,
        'interval': interval,
        'startup_delay': startup_delay,
        'capture_output': capture_output,
        'output_log': output_log,
        'no_tui': no_tui,
        'verbose': verbose
    }


@click.command()
@click.option('--process-name', '-p', help='Name of the process to monitor (e.g., notepad.exe)')
@click.option('--executable', '-e', type=click.Path(), help='Path to executable to launch')
@click.option('--interval', '-i', default=5.0, type=float, help='Check interval in seconds (default: 5.0)')
@click.option('--startup-delay', '-d', default=2.0, type=float, help='Delay after launching before resuming checks (default: 2.0)')
@click.option('--capture-output', '-c', is_flag=True, help='Capture stdout/stderr from the process')
@click.option('--output-log', '-o', type=click.Path(), help='Log file for captured output')
@click.option('--no-tui', is_flag=True, help='Run without TUI (headless mode)')
@click.option('--verbose', '-v', is_flag=True, help='Enable verbose logging')
@click.option('--config', type=click.Path(), help='YAML configuration file to load')
def main(process_name, executable, interval, startup_delay, capture_output, output_log, no_tui, verbose, config):
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
        'no_tui': no_tui or config_data.get('no_tui', False),
        'verbose': verbose or config_data.get('verbose', False)
    }
    
    # Validate required parameters
    if not final_config['process_name']:
        raise click.ClickException("Process name is required (--process-name or in config file)")
    if not final_config['executable']:
        raise click.ClickException("Executable path is required (--executable or in config file)")
    
    # Validate executable exists
    executable_path = Path(final_config['executable'])
    if not executable_path.exists():
        raise click.ClickException(f"Executable not found: {executable_path}")
    
    # Setup logging
    log_level = logging.DEBUG if final_config['verbose'] else logging.INFO
    logging.basicConfig(
        level=log_level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            RichHandler(rich_tracebacks=True) if not final_config['no_tui'] else logging.StreamHandler(),
            logging.FileHandler('process_monitor.log')
        ]
    )
    
    # Save current configuration as default-config.yaml
    save_config('default-config.yaml', get_default_config(**final_config))
    
    # Create monitor
    monitor = ProcessMonitor(
        process_name=final_config['process_name'],
        executable_path=final_config['executable'],
        check_interval=final_config['interval'],
        startup_delay=final_config['startup_delay'],
        capture_output=final_config['capture_output'],
        output_log=final_config['output_log']
    )
    
    if final_config['no_tui']:
        # Run in headless mode
        console = Console()
        console.print(f"[green]Starting process monitor v{VERSION} in headless mode...[/green]")
        console.print(f"Monitoring: {final_config['process_name']}")
        console.print(f"Executable: {final_config['executable']}")
        console.print(f"Check interval: {final_config['interval']}s")
        
        try:
            monitor.monitor_loop()
        except KeyboardInterrupt:
            console.print("\n[yellow]Stopping monitor...[/yellow]")
            monitor.stop()
    else:
        # Run with TUI
        tui = MonitorTUI(monitor)
        tui.run()


if __name__ == '__main__':
    main()