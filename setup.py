from cx_Freeze import setup, Executable

# Dependencies are automatically detected, but it might need
# fine tuning.
build_options = {'packages': [], 'excludes': []}

base = 'console'

executables = [
    Executable('process-monitor.py', base=base)
]

setup(name='process-monitor',
      version = '1.0',
      description = 'Monitor and relaunch a process',
      options = {'build_exe': build_options},
      executables = executables)
