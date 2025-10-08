"""
Build script for Process Monitor
Creates standalone executable using PyInstaller

Usage: python build.py
"""

import subprocess
import sys
import shutil
from pathlib import Path


def check_pyinstaller():
    """Check if PyInstaller is installed"""
    try:
        import PyInstaller
        print(f"✓ PyInstaller {PyInstaller.__version__} found")
        return True
    except ImportError:
        print("✗ PyInstaller not found")
        print("\nPlease install it with:")
        print("  pip install pyinstaller")
        return False


def clean_build_dirs():
    """Clean previous build directories"""
    dirs_to_clean = ['build', 'dist', '__pycache__']
    for dir_name in dirs_to_clean:
        dir_path = Path(dir_name)
        if dir_path.exists():
            print(f"Cleaning {dir_name}/")
            shutil.rmtree(dir_path)
    
    # Clean spec file
    spec_file = Path('process_monitor.spec')
    if spec_file.exists():
        print("Cleaning process_monitor.spec")
        spec_file.unlink()


def build_executable():
    """Build the executable using PyInstaller"""
    print("\n" + "="*50)
    print("Building Process Monitor Executable")
    print("="*50 + "\n")
    
    # PyInstaller command
    cmd = [
        'pyinstaller',
        '--onefile',  # Single executable file
        '--windowed' if sys.platform == 'win32' else '--console',  # Console mode
        '--name=process-monitor',  # Output name
        '--clean',  # Clean cache
        '--noconfirm',  # Overwrite without asking
        # Hidden imports that PyInstaller might miss
        '--hidden-import=rich',
        '--hidden-import=rich.console',
        '--hidden-import=rich.live',
        '--hidden-import=rich.layout',
        '--hidden-import=rich.panel',
        '--hidden-import=rich.table',
        '--hidden-import=rich.text',
        '--hidden-import=rich.logging',
        '--hidden-import=psutil',
        '--hidden-import=click',
        '--hidden-import=yaml',
        'process_monitor.py'
    ]
    
    print("Running PyInstaller with options:")
    print("  " + " ".join(cmd))
    print()
    
    try:
        result = subprocess.run(cmd, check=True)
        print("\n" + "="*50)
        print("✓ Build successful!")
        print("="*50)
        
        # Show output location
        if sys.platform == 'win32':
            exe_path = Path('dist/process-monitor.exe')
        else:
            exe_path = Path('dist/process-monitor')
        
        if exe_path.exists():
            print(f"\nExecutable created: {exe_path.absolute()}")
            print(f"Size: {exe_path.stat().st_size / (1024*1024):.2f} MB")
            
            print("\nUsage examples:")
            if sys.platform == 'win32':
                print('  .\\dist\\process-monitor.exe -p notepad.exe -e "C:\\Windows\\System32\\notepad.exe"')
                print('  .\\dist\\process-monitor.exe -p myapp.exe -e myapp.exe -c -o output.log')
                print('  .\\dist\\process-monitor.exe --config my-config.yaml')
            else:
                print('  ./dist/process-monitor -p firefox -e /usr/bin/firefox')
                print('  ./dist/process-monitor -p myapp -e ./myapp -c -o output.log')
                print('  ./dist/process-monitor --config my-config.yaml')
        
        return True
        
    except subprocess.CalledProcessError as e:
        print("\n" + "="*50)
        print("✗ Build failed!")
        print("="*50)
        print(f"\nError: {e}")
        return False
    except Exception as e:
        print(f"\n✗ Unexpected error: {e}")
        return False


def main():
    """Main build process"""
    print("Process Monitor - Build Script")
    print("="*50 + "\n")
    
    # Check dependencies
    if not check_pyinstaller():
        sys.exit(1)
    
    # Check if source file exists
    if not Path('process_monitor.py').exists():
        print("✗ Error: process_monitor.py not found in current directory")
        sys.exit(1)
    
    print("✓ Source file found: process_monitor.py\n")
    
    # Ask for confirmation
    response = input("Clean previous build directories? [Y/n]: ").strip().lower()
    if response != 'n':
        clean_build_dirs()
        print()
    
    # Build
    success = build_executable()
    
    if success:
        print("\n✓ Done! Your executable is ready in the dist/ directory")
        sys.exit(0)
    else:
        print("\n✗ Build failed. Please check the errors above.")
        sys.exit(1)


if __name__ == '__main__':
    main()