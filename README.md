# Process Monitor v1.3.0

A cross-platform process monitoring tool that automatically relaunches applications when they stop running. Features a beautiful terminal UI, output capture, file operations, and comprehensive configuration options.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
  - [Option 1: Run from Source](#option-1-run-from-source)
  - [Option 2: Build Standalone Executable](#option-2-build-standalone-executable)
- [Usage](#usage)
  - [Basic Usage](#basic-usage)
  - [With Output Capture (TUI)](#with-output-capture-tui)
  - [With Output Capture (Log File)](#with-output-capture-log-file)
  - [Headless Mode](#headless-mode)
  - [TUI Controls (Textual)](#tui-controls-textual)
  - [Custom Intervals](#custom-intervals)
  - [Using Configuration Files](#using-configuration-files)
- [YAML Configuration](#yaml-configuration)
  - [Configuration Format](#configuration-format)
  - [File Operations Syntax](#file-operations-syntax)
  - [Configuration Example](#configuration-example)
  - [Configuration Priority](#configuration-priority)
- [Examples](#examples)
  - [Windows](#windows)
  - [Linux/macOS](#linuxmacos)
- [Building Executable](#building-executable)
  - [Manual cx_Freeze Build](#manual-cx_freeze-build)
- [Running as a Service](#running-as-a-service)
  - [Windows (using NSSM)](#windows-using-nssm)
  - [Linux (using systemd)](#linux-using-systemd)
- [Logging](#logging)
  - [Log Cleanup](#log-cleanup)
- [Dependencies](#dependencies)
- [Troubleshooting](#troubleshooting)
- [License](#license)
- [Contributing](#contributing)

## Features

- ✅ **Cross-platform** - Works on Windows, Linux, and macOS
- ✅ **Automatic relaunching** - Monitors process and restarts when not found (configurable)
- ✅ **Beautiful TUI** - Real-time status display with Textual framework
- ✅ **Output capture** - Redirect STDOUT/STDERR to TUI or log file with timestamps
- ✅ **YAML configuration** - Save and load settings from config files
- ✅ **Headless mode** - Run as background service with rich progress display
- ✅ **File operations** - Perform file copies and cleanup before/after process execution
- ✅ **Lock file support** - Wait for lock files after process launch
- ✅ **Working directory** - Set custom working directory for launched processes
- ✅ **Process management** - Kill existing processes, disable monitoring, exit controls
- ✅ **Log management** - Automatic cleanup of old log files
- ✅ **Configurable** - Extensive customization via CLI or YAML
- ✅ **Standalone executable** - Build with cx_Freeze

## Installation

### Option 1: Run from Source

```bash
# Install dependencies
pip install -r requirements.txt

# Run the monitor
python main.py --process-name notepad.exe --executable "C:\Windows\System32\notepad.exe"
```

### Option 2: Build Standalone Executable

```bash
# Install all dependencies including cx_Freeze
pip install -r requirements.txt
pip install cx_Freeze

# Build the executable
python setup.py build

# Run the executable (Windows example)
build/exe.win-amd64-3.12/process-monitor.exe --process-name notepad.exe --executable notepad.exe
```

## Usage

### Basic Usage

Monitor a process and relaunch if not found:

```bash
python main.py --process-name notepad.exe --executable "C:\Windows\System32\notepad.exe"
```

### With Output Capture (TUI)

Capture and display STDOUT/STDERR in the TUI:

```bash
python main.py --process-name myapp.exe --executable "C:\path\to\myapp.exe" --capture-output
```

### With Output Capture (Log File)

Capture output to a log file with timestamp:

```bash
python main.py --process-name myapp.exe --executable myapp.exe --capture-output --output-log output.log
```

### Headless Mode

Run without TUI (good for services):

```bash
python main.py --process-name myapp.exe --executable myapp.exe --no-tui --capture-output --output-log output.log
```

### TUI Controls (Textual)

- Press `q` to quit the TUI
- Press `k` to kill existing processes and quit
- Press `Ctrl+C` to interrupt and stop monitoring

### Custom Intervals

Set custom check interval (0.5s) and startup delay (2s):

```bash
python main.py --process-name myapp.exe --executable myapp.exe --interval 0.5 --startup-delay 2.0
```

### Using Configuration Files

Create a YAML configuration file with your settings:

**config.yaml:**
```yaml
process_name: "notepad.exe"
executable: "C:\\Windows\\System32\\notepad.exe"
interval: 0.5
startup_delay: 2.0
capture_output: true
output_log: "notepad_output.log"
tui: true
verbose: true
kill_existing: false
monitor_process: true
```

Then run with the config file:

```bash
python main.py --config config.yaml
```

**CLI options override config file values:**

```bash
# Use config file but override the interval
python main.py --config config.yaml --interval 1.0
```

## YAML Configuration

The process monitor can load settings from YAML configuration files using the `--config` option. This makes it easy to save and reuse monitoring configurations.

### Configuration Format

```yaml
# Required settings
process_name: "myapp.exe"           # Process name to monitor
executable: "/path/to/executable"   # Path to executable to launch

# Timing settings
interval: 0.5                       # Check interval in seconds (default: 0.5)
startup_delay: 2.0                  # Delay after launching before resuming checks (default: 2.0)

# Output settings
capture_output: false               # Capture stdout/stderr (default: false)
output_log: "output.log"            # Log file for captured output (optional)

# Environment settings
working_dir: "/path/to/work/dir"     # Working directory for process (optional)

# UI settings
tui: true                           # Use TUI mode (default: true)

# Logging settings
verbose: false                      # Enable verbose logging (default: false)

# Process management
kill_existing: false                # Kill existing processes before launching (default: false)
monitor_process: true               # Auto-relaunch when not found (default: true)

# Synchronization
lock_file: "/path/to/lock"           # Wait for lock file after launch (optional)

# File operations
initial_file_copies:                # Files to copy before launch
  - source: "/path/to/src"
    destination: "/path/to/dst"
final_file_copies:                  # Files to copy after stop
  - source: "/path/to/src"
    destination: "/path/to/dst"
initial_file_cleanup:               # Files to delete before launch
  - "/path/to/file"
final_file_cleanup:                 # Files to delete after stop
  - "/path/to/file"

# Exit behavior
exit_delay: 0.0                     # Delay before exiting (default: 0.0)
wait_keypress: false                # Wait for keypress before exit (default: false)

# Log management
log_cleanup_enabled: false          # Enable automatic log cleanup (default: false)
log_cleanup_keep_count: 3           # Number of recent logs to keep (default: 3)
```

### File Operations Syntax

File operations use source-destination pairs for copies and simple paths for cleanup:

```yaml
# Copy operations
initial_file_copies:
  - source: "/path/to/config.template"
    destination: "/path/to/config.ini"
  - source: "/path/to/data.db"
    destination: "/path/to/backup/data.db"

final_file_copies:
  - source: "/path/to/log.txt"
    destination: "/path/to/archive/log.txt"

# Cleanup operations
initial_file_cleanup:
  - "/tmp/old_config.ini"
  - "/tmp/cache/"

final_file_cleanup:
  - "/tmp/session_data"
```

### Configuration Example

A comprehensive configuration example is provided in `config.example.yaml`. Copy this file to `config.yaml` and modify the values for your specific use case.

### Configuration Priority

When both CLI options and a config file are provided:
1. CLI options take precedence over config file values
2. Config file values are used when CLI options are not specified
3. Built-in defaults are used when neither CLI nor config specify a value

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | YAML configuration file to load | None |
| `--process-name` | Name of process to monitor | Required* |
| `--executable` | Path to executable to launch | Required* |
| `--interval` | Check interval in seconds | 0.5 |
| `--startup-delay` | Delay after launching before resuming checks | 2.0 |
| `--capture-output` | Capture STDOUT/STDERR from process | False |
| `--output-log` | Log file path for captured output | None |
| `--working-dir` | Working directory for the launched process | None |
| `--tui` / `--no-tui` | Use TUI mode | True |
| `--verbose` | Enable verbose logging | False |
| `--kill-existing` | Kill existing processes with same name before launching | False |
| `--monitor` / `--no-monitor` | Auto-relaunch missing process | True |
| `--lock-file` | Path to lock file to wait for after launching | None |
| `--exit-delay` | Delay in seconds before exiting | 0.0 |
| `--wait-keypress` | Wait for keypress before exiting | False |
| `--log-cleanup` / `--no-log-cleanup` | Enable automatic cleanup of old log files | False |
| `--log-cleanup-keep-count` | Number of recent log files to keep | 3 |

*Required unless provided in config file

## Examples

### Windows

Monitor Notepad:

```bash
python main.py --process-name notepad.exe --executable "C:\Windows\System32\notepad.exe"
```

Monitor custom application with output capture:

```bash
python main.py --process-name myserver.exe --executable "C:\Apps\myserver.exe" --capture-output --output-log server.log
```

Monitor with file operations:

```bash
python main.py --config config.yaml
```

### Linux/macOS

Monitor Firefox:

```bash
python main.py --process-name firefox --executable /usr/bin/firefox
```

Monitor custom application:

```bash
python main.py --process-name myapp --executable /opt/myapp/myapp --capture-output --output-log output.log
```

Monitor with working directory and lock file:

```bash
python main.py --process-name myapp --executable /opt/myapp/myapp --working-dir /opt/myapp --lock-file /tmp/myapp.lock
```

## Building Executable

The `setup.py` script simplifies creating a standalone executable using cx_Freeze:

```bash
# Make sure cx_Freeze is installed
pip install cx_Freeze

# Run the build script
python setup.py build
```

The script will create an executable in `build/exe.<platform>-<python_version>/process-monitor` (or `.exe` on Windows).

### Manual cx_Freeze Build

If you prefer to modify the build:

```python
# setup.py contents
from cx_Freeze import setup, Executable

build_options = {
    'packages': [],
    'excludes': ['tkinter', 'unittest', 'email', 'http', 'xml', 'pyinstaller', 'cx_Freeze', 'setuptools', 'wheel']
}

executables = [
    Executable('main.py', base=None, target_name='process-monitor.exe')
]

setup(
    name='process-monitor',
    version='1.3.0',
    description='Monitor and relaunch a process',
    options={'build_exe': build_options},
    executables=executables
)
```

## Running as a Service

### Windows (using NSSM)

```bash
# Download NSSM from https://nssm.cc/
nssm install ProcessMonitor "C:\path\to\process-monitor.exe" --process-name myapp.exe --executable myapp.exe --no-tui --capture-output --output-log output.log
nssm start ProcessMonitor
```

### Linux (using systemd)

Create `/etc/systemd/system/process-monitor.service`:

```ini
[Unit]
Description=Process Monitor
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/process-monitor --process-name myapp --executable /opt/myapp/myapp --no-tui --capture-output --output-log /var/log/myapp.log
Restart=always
User=youruser

[Install]
WantedBy=multi-user.target
```

Then:

```bash
sudo systemctl daemon-reload
sudo systemctl enable process-monitor
sudo systemctl start process-monitor
```

## Logging

The monitor creates timestamped log files:

- `process_monitor-YYYYMMDDHHMMSS.log` - Monitor's own activity log
- Custom output log (if `--output-log` specified) - Captured process STDOUT/STDERR with timestamps

### Log Cleanup

Enable automatic cleanup of old logs:

```bash
python main.py --log-cleanup --log-cleanup-keep-count 5 --config config.yaml
```

This keeps only the 5 most recent log files of each type.

## Dependencies

- **psutil** (≥7.0.0) - Cross-platform process utilities
- **PyYAML** (≥6.0.0) - YAML configuration file support
- **rich** (≥14.0.0) - Terminal formatting and headless output
- **typer** (≥0.19.0) - Command-line interface
- **textual** (≥0.57.0) - TUI framework
- **cx_Freeze** (optional) - For building executables

## Troubleshooting

### Process not detected

Make sure the process name matches exactly (including `.exe` on Windows). Use Task Manager or `ps aux` to verify the exact process name.

### Executable won't run

Make sure you're using the full path to the executable, or the executable is in your PATH.

### Output not captured

Some applications redirect their output in ways that can't be captured. Try with a simple application first to verify functionality.

### Permission errors

On Linux/macOS, you may need to make files executable:

```bash
chmod +x build/exe.linux-x86_64-3.12/process-monitor
chmod +x /path/to/your/app
```

### File operations fail

Ensure the user running the monitor has permissions to read/write/delete the specified files and directories.

### Lock file not found

The monitor will wait indefinitely for the lock file. Ensure your application creates the lock file during startup.

## License

MIT License - Feel free to use and modify as needed.

## Contributing

Contributions welcome! Please feel free to submit issues or pull requests.
