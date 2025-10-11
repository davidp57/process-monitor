import logging
import queue
import time
from datetime import datetime
from typing import Optional

from rich import print as rprint
from rich.console import Console
from rich.progress import Progress, SpinnerColumn, TextColumn

from constants import LOG_QUEUE_SIZE
from process_monitor import ProcessMonitor
from tui import MonitorTextualApp, TUIHandler


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