import contextlib
import logging
import queue
import signal
import threading

from rich.panel import Panel
from rich.table import Table
from rich.text import Text

# Textual TUI
from textual.app import App, ComposeResult
from textual.containers import Vertical
from textual.widgets import Footer, Header, Log, Static

from constants import VERSION, MAX_LOG_LINES, LOG_QUEUE_SIZE
from process_monitor import ProcessMonitor


class TUIHandler(logging.Handler):
    """Logging handler that queues log messages for TUI display."""

    def __init__(self, queue):
        super().__init__()
        self.log_queue = queue

    def emit(self, record):
        msg = self.format(record)
        with contextlib.suppress(queue.Full):
            self.log_queue.put_nowait(msg)


class MonitorTextualApp(App):
    """Textual-based TUI for the ProcessMonitor."""
    TITLE = f"Process Monitor {VERSION}"
    BINDINGS = [("q", "quit", "Quit"),("k", "kill", "Kill and quit"), ("p", "toggle_pause", "Toggle pause")]

    def __init__(self, monitor: ProcessMonitor, log_queue: queue.Queue, exit_delay: float, wait_keypress: bool):
        """Initialize the Textual TUI app with monitor and configuration."""
        super().__init__()
        self.monitor = monitor
        self.log_queue = log_queue
        self.exit_delay = exit_delay
        # wait_keypress is not used in TUI mode
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
        if self.monitor.process_status == "Paused":
            status_color = "orange_red1"
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
            if self.exit_delay > 0:
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
            self.monitor.stop()
            self.exiting = True
            if self.exit_delay > 0:
                self.exit_static.update(f"Exiting in {self.exit_delay} seconds")
                self.set_timer(self.exit_delay, self.exit)
            else:
                self.exit()

    def action_kill(self) -> None:
        """Kill existing processes and exit the application."""
        self.monitor.kill_existing_processes()
        self.action_quit()

    def action_toggle_pause(self) -> None:
        """Toggle pause/unpause the running process."""
        if not self.monitor.is_process_running():
            self.logger.warning("Cannot pause/unpause: process is not running")
            return
        if self.monitor.paused:
            self.monitor.unpause_process()
        else:
            self.monitor.pause_process()

    def handle_exit(self) -> None:
        """Kill existing processes if configured to do so on exit."""
        if self.monitor.kill_existing:
            self.monitor.kill_existing_processes()