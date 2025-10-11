# Process Monitor

A cross-platform process monitoring tool that automatically relaunches applications when they stop running. Features a beautiful terminal UI and output capture capabilities.

## Features

- ✅ **Cross-platform** - Works on Windows, Linux, and macOS
- ✅ **Automatic relaunching** - Monitors process and restarts when not found
- ✅ **Beautiful TUI** - Real-time status display with Textual (Rich rendering)
- ✅ **Output capture** - Redirect STDOUT/STDERR to TUI or log file
- ✅ **YAML configuration** - Save and load settings from config files
- ✅ **Headless mode** - Run as background service without TUI
- ✅ **Configurable** - Customize check intervals and delays
- ✅ **Standalone executable** - Build with PyInstaller

## Installation

### Option 1: Run from Source

```bash
# Install dependencies
pip install -r requirements.txt

# Run the monitor
python process-monitor.py -p notepad.exe -e "C:\Windows\System32\notepad.exe"
```

### Option 2: Build Standalone Executable

```bash
# Install all dependencies including PyInstaller
pip install -r requirements.txt

# Build the executable
python build.py

# Run the executable
./dist/process-monitor -p notepad.exe -e notepad.exe
```

## Usage

### Basic Usage

Monitor a process and relaunch if not found:

```bash
python process-monitor.py -p notepad.exe -e "C:\Windows\System32\notepad.exe"
```

### With Output Capture (TUI)

Capture and display STDOUT/STDERR in the TUI:

```bash
python process-monitor.py -p myapp.exe -e "C:\path\to\myapp.exe" -c
```

### With Output Capture (Log File)

Capture output to a log file:

```bash
python process-monitor.py -p myapp.exe -e myapp.exe -c -o output.log
```

### Headless Mode

Run without TUI (good for services):

```bash
python process-monitor.py -p myapp.exe -e myapp.exe --no-tui -c -o output.log
```

### TUI Controls (Textual)

- Press `q` to quit the TUI
- Press `Ctrl+C` to interrupt and stop monitoring

### Custom Intervals

Set custom check interval (2.5s) and startup delay (5s):

```bash
python process-monitor.py -p myapp.exe -e myapp.exe -i 2.5 -d 5.0
```

### Using Configuration Files

Create a YAML configuration file with your settings:

**config.yaml:**
```yaml
process_name: "notepad.exe"
executable: "C:\\Windows\\System32\\notepad.exe"
interval: 3.0
startup_delay: 1.0
capture_output: true
output_log: "notepad_output.log"
no_tui: false
verbose: true
```

Then run with the config file:

```bash
python process-monitor.py --config config.yaml
```

**CLI options override config file values:**

```bash
# Use config file but override the interval
python process-monitor.py --config config.yaml --interval 10.0
```

## YAML Configuration

The process monitor can load settings from YAML configuration files using the `--config` option. This makes it easy to save and reuse monitoring configurations.

### Configuration Format

```yaml
# Process to monitor
process_name: "myapp.exe"           # Required: Process name to monitor
executable: "/path/to/executable"   # Required: Path to executable to launch

# Timing settings
interval: 5.0                       # Check interval in seconds (default: 5.0)
startup_delay: 2.0                  # Delay after launch in seconds (default: 2.0)

# Output settings
capture_output: false               # Capture stdout/stderr (default: false)
output_log: "output.log"            # Log file for captured output (optional)

# Display settings
no_tui: false                       # Run without TUI (default: false)
verbose: false                      # Enable verbose logging (default: false)
kill_existing: false                # Kill existing processes before launching (default: false)
```

### Configuration Example

A comprehensive configuration example is provided in `config.example.yaml`. Copy this file to `config.yaml` and modify the values for your specific use case.

### Configuration Priority

When both CLI options and a config file are provided:
1. CLI options take precedence over config file values
2. Config file values are used when CLI options are not specified
3. Built-in defaults are used when neither CLI nor config specify a value

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--process-name` | `-p` | Name of process to monitor (e.g., `notepad.exe`) | Required* |
| `--executable` | `-e` | Path to executable to launch | Required* |
| `--interval` | `-i` | Check interval in seconds | 5.0 |
| `--startup-delay` | `-d` | Delay after launching before resuming checks | 2.0 |
| `--capture-output` | `-c` | Capture STDOUT/STDERR from process | False |
| `--output-log` | `-o` | Log file path for captured output | None |
| `--no-tui` | | Run without TUI (headless mode) | False |
| `--verbose` | `-v` | Enable verbose logging | False |
| `--kill-existing` | | Kill existing processes with same name and executable before launching | False |
| `--config` | | YAML configuration file to load | None |

*Required unless provided in config file

## Examples

### Windows

Monitor Notepad:

```bash
python process-monitor.py -p notepad.exe -e "C:\Windows\System32\notepad.exe"
```

Monitor custom application with output capture:

```bash
python process-monitor.py -p myserver.exe -e "C:\Apps\myserver.exe" -c -o server.log
```

### Linux/macOS

Monitor Firefox:

```bash
python process-monitor.py -p firefox -e /usr/bin/firefox
```

Monitor custom application:

```bash
python process-monitor.py -p myapp -e /opt/myapp/myapp -c -o output.log
```

## Building Executable

The `build.py` script simplifies creating a standalone executable:

```bash
# Make sure PyInstaller is installed
pip install pyinstaller

# Run the build script
python build.py
```

The script will:

1. Check for PyInstaller installation
2. Clean previous build directories
3. Build a single executable file
4. Display the output location and size

The executable will be created in `dist/process-monitor` (or `dist/process-monitor.exe` on Windows).

### Manual PyInstaller Build

If you prefer to build manually:

```bash
pyinstaller --onefile --console --name=process-monitor \
    --hidden-import=rich --hidden-import=textual --hidden-import=psutil --hidden-import=click \
    process-monitor.py
```

## Running as a Service

### Windows (using NSSM)

```bash
# Download NSSM from https://nssm.cc/
nssm install ProcessMonitor "C:\path\to\process-monitor.exe" -p myapp.exe -e myapp.exe --no-tui -c -o output.log
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
ExecStart=/usr/local/bin/process-monitor -p myapp -e /opt/myapp/myapp --no-tui -c -o /var/log/myapp.log
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

The monitor creates two log files:

- `process-monitor.log` - Monitor's own activity log
- Custom output log (if `-o` specified) - Captured process STDOUT/STDERR

## Dependencies

- **psutil** (≥7.0.0) - Cross-platform process utilities
- **rich** (≥14.0.0) - Terminal formatting and headless output
- **textual** (≥0.57.0) - TUI framework
- **typer** (≥0.19.0) - Command-line interface
- **pyyaml** (≥6.0.0) - YAML configuration file support
- **pyinstaller** (≥6.0.0) - For building executables (optional)

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
chmod +x dist/process-monitor
chmod +x /path/to/your/app
```

## License

MIT License - Feel free to use and modify as needed.

## Contributing

Contributions welcome! Please feel free to submit issues or pull requests.
