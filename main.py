import typer
from datetime import datetime
from pathlib import Path
from typing import Optional

from config import get_default_config, load_and_merge_config, validate_config
from logging_utils import setup_logging, cleanup_old_logs
from process_monitor import ProcessMonitor
from run_modes import run_headless_mode, run_tui_mode


app = typer.Typer()


@app.callback(invoke_without_command=True)
def main(
    config: Optional[str] = typer.Option(None, help='YAML configuration file to load'),
    process_name: Optional[str] = typer.Option(None, help='Name of the process to monitor (e.g., notepad.exe)'),
    executable: Optional[str] = typer.Option(None, help='Path to executable to launch'),
    interval: Optional[float] = typer.Option(None, help='Check interval in seconds (default: 0.5)'),
    startup_delay: Optional[float] = typer.Option(None, help='Delay after launching before resuming checks (default: 2.0)'),
    capture_output: Optional[bool] = typer.Option(None, help='Capture stdout/stderr from the process'),
    output_log: Optional[str] = typer.Option(None, help='Log file for captured output'),
    working_dir: Optional[str] = typer.Option(None, help='Working directory for the launched process'),
    tui: Optional[bool] = typer.Option(None, '--tui/--no-tui', help='Use TUI mode'),
    verbose: Optional[bool] = typer.Option(None, help='Enable verbose logging'),
    kill_existing: Optional[bool] = typer.Option(None, help='Kill existing processes with same name and executable before launching and after exiting'),
    monitor: Optional[bool] = typer.Option(None, '--monitor/--no-monitor', help='Relaunch missing process automatically (default: enabled)'),
    lock_file: Optional[str] = typer.Option(None, help='Path to lock file to wait for after launching the process'),
    exit_delay: Optional[float] = typer.Option(None, help='Delay in seconds before exiting the app'),
    wait_keypress: Optional[bool] = typer.Option(None, help='Wait for keypress before exiting'),
    log_cleanup: Optional[bool] = typer.Option(None, '--log-cleanup/--no-log-cleanup', help='Enable automatic cleanup of old log files on startup'),
    log_cleanup_keep_count: Optional[int] = typer.Option(None, help='Number of recent log files to keep when cleanup is enabled (default: 3)'),
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
        'log_cleanup_enabled': log_cleanup,
        'log_cleanup_keep_count': log_cleanup_keep_count,
    }
    final_config = load_and_merge_config(config, cli_args)
    validate_config(final_config)

    # Setup logging
    setup_logging(final_config['verbose'], final_config['tui'])

    # Clean up old log files if enabled
    if final_config['log_cleanup_enabled']:
        cleanup_old_logs(final_config['output_log'], final_config['log_cleanup_keep_count'])

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
        initial_file_cleanup=final_config['initial_file_cleanup'],
        final_file_cleanup=final_config['final_file_cleanup'],
    )

    # Run with TUI if requested
    if final_config['tui']:
        run_tui_mode(monitor_obj, final_config)
    else:
        run_headless_mode(monitor_obj, final_config, output_log_with_timestamp)


if __name__ == '__main__':
    app()