import logging
from typing import Optional

import typer
import yaml


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


def load_and_merge_config(config_file: Optional[str], cli_args: dict) -> dict:
    """Load config from file and merge with CLI args, CLI takes precedence"""
    config_data = load_config(config_file) if config_file else {}
    return {
        'process_name': cli_args.get('process_name') or config_data.get('process_name'),
        'executable': cli_args.get('executable') or config_data.get('executable'),
        'interval': cli_args.get('interval', config_data.get('interval', 0.5)),
        'startup_delay': cli_args.get('startup_delay', config_data.get('startup_delay', 2.0)),
        'capture_output': cli_args.get('capture_output', config_data.get('capture_output', False)),
        'output_log': cli_args.get('output_log') or config_data.get('output_log'),
        'working_dir': cli_args.get('working_dir') or config_data.get('working_dir'),
        'tui': cli_args.get('tui', config_data.get('tui', True)),
        'verbose': cli_args.get('verbose', config_data.get('verbose', False)),
        'kill_existing': cli_args.get('kill_existing', config_data.get('kill_existing', False)),
        'monitor_process': cli_args.get('monitor', config_data.get('monitor_process', True)),
        'lock_file': cli_args.get('lock_file') or config_data.get('lock_file'),
        'initial_file_copies': config_data.get('initial_file_copies', []),
        'final_file_copies': config_data.get('final_file_copies', []),
        'exit_delay': cli_args.get('exit_delay', config_data.get('exit_delay', 0.0)),
        'wait_keypress': cli_args.get('wait_keypress', config_data.get('wait_keypress', False)),
    }


def validate_config(config: dict):
    """Validate configuration parameters"""
    if not config['process_name']:
        typer.echo("Process name is required (--process-name or in config file)", err=True)
        raise typer.Exit(1)
    if not config['executable']:
        typer.echo("Executable path is required (--executable or in config file)", err=True)
        raise typer.Exit(1)
    if config['interval'] <= 0:
        typer.echo("Check interval must be positive", err=True)
        raise typer.Exit(1)
    if config['startup_delay'] < 0:
        typer.echo("Startup delay must be non-negative", err=True)
        raise typer.Exit(1)


def get_default_config(process_name: Optional[str] = None, executable: Optional[str] = None,
                       interval: float = 0.5, startup_delay: float = 2.0,
                       capture_output: bool = False, output_log: Optional[str] = None,
                       working_dir: Optional[str] = None, tui: bool = True,
                       verbose: bool = False, kill_existing: bool = False,
                       monitor_process: bool = True, lock_file: Optional[str] = None,
                       initial_file_copies: Optional[list] = None, final_file_copies: Optional[list] = None,
                       exit_delay: float = 0.0, wait_keypress: bool = False) -> dict:
    """Get default configuration dictionary"""
    config = {
        'process_name': process_name,
        'executable': executable,
        'interval': interval,
        'startup_delay': startup_delay,
        'capture_output': capture_output,
        'tui': tui,
        'verbose': verbose,
        'kill_existing': kill_existing,
        'monitor_process': monitor_process,
        'exit_delay': exit_delay,
        'wait_keypress': wait_keypress,
    }
    # Only include optional fields if they have values
    if output_log:
        config['output_log'] = output_log
    if working_dir:
        config['working_dir'] = working_dir
    if lock_file:
        config['lock_file'] = lock_file
    if initial_file_copies:
        config['initial_file_copies'] = initial_file_copies
    if final_file_copies:
        config['final_file_copies'] = final_file_copies
    return config