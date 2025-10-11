import logging
from datetime import datetime

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