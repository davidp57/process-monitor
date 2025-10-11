import logging
from datetime import datetime
from pathlib import Path
from typing import Optional

from rich.logging import RichHandler


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


def cleanup_old_logs(output_log_base: Optional[str], keep_count: int = 3):
    """Clean up old log files, keeping only the most recent ones"""
    if keep_count <= 0:
        return

    # Clean up process monitor logs (process_monitor-*.log)
    current_dir = Path('.')
    pm_logs = list(current_dir.glob('process_monitor-*.log'))
    if len(pm_logs) > keep_count:
        # Sort by modification time, newest first
        pm_logs.sort(key=lambda x: x.stat().st_mtime, reverse=True)
        for old_log in pm_logs[keep_count:]:
            try:
                old_log.unlink()
                logging.info(f"Cleaned up old process monitor log: {old_log}")
            except Exception as e:
                logging.warning(f"Failed to delete old log {old_log}: {e}")

    # Clean up output logs if base path is provided
    if output_log_base:
        _do_cleanup_old_logs(output_log_base, keep_count)


def _do_cleanup_old_logs(output_log_base, keep_count):
    output_path = Path(output_log_base)
    output_dir = output_path.parent
    stem = output_path.stem
    suffix = output_path.suffix

    # Find all files matching stem-*-timestamp*suffix pattern
    pattern = f"{stem}-*{suffix}"
    output_logs = list(output_dir.glob(pattern))
    if len(output_logs) > keep_count:
        # Sort by modification time, newest first
        output_logs.sort(key=lambda x: x.stat().st_mtime, reverse=True)
        for old_log in output_logs[keep_count:]:
            try:
                old_log.unlink()
                logging.info(f"Cleaned up old output log: {old_log}")
            except Exception as e:
                logging.warning(f"Failed to delete old output log {old_log}: {e}")