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

from constants import MAX_OUTPUT_LINES, OUTPUT_QUEUE_SIZE


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