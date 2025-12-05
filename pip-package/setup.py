#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0

import sys
from pathlib import Path
from setuptools import setup
from setuptools.command.install import install

# Get the package directory
PACKAGE_DIR = Path(__file__).parent


class PostInstallCommand(install):
    """Post-installation command to download the binary."""
    def run(self):
        install.run(self)
        # Run the install script to download the binary
        # We need to run it after installation so the package is available
        install_script = PACKAGE_DIR / "shaper_pkg" / "install.py"
        if install_script.exists():
            import subprocess
            try:
                # Add the installed package to the path
                result = subprocess.run(
                    [sys.executable, str(install_script)],
                    cwd=str(PACKAGE_DIR),
                    capture_output=True,
                    text=True
                )
                if result.returncode != 0:
                    print(f"Warning: Failed to download binary: {result.stderr}", file=sys.stderr)
                    print("The binary will be downloaded automatically on first use.", file=sys.stderr)
                else:
                    print(result.stdout, end="")
            except Exception as e:
                print(f"Warning: Failed to download binary: {e}", file=sys.stderr)
                print("The binary will be downloaded automatically on first use.", file=sys.stderr)


if __name__ == "__main__":
    setup(
        cmdclass={
            "install": PostInstallCommand,
        },
    )
