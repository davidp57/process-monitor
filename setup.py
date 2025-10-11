from cx_Freeze import setup, Executable
import sys

# Set base to Win32GUI to suppress console window
base = None

build_options = {
    'packages': [],
    'excludes': ['tkinter', 'unittest', 'email', 'http', 'xml', 'pyinstaller', 'cx_Freeze', 'setuptools', 'wheel']
}

executables = [
    Executable('process-monitor.py', base=base, target_name='process-monitor.exe')
]

setup(
    name='process-monitor',
    version='1.0',
    description='Monitor and relaunch a process',
    options={'build_exe': build_options},
    executables=executables
)