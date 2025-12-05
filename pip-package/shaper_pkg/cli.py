#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0

"""Command-line interface for Shaper."""

import os
import sys
from pathlib import Path

# Get package directory
PACKAGE_DIR = Path(__file__).parent
BINARY_PATH = PACKAGE_DIR / "bin" / "shaper"


def main():
    """Main entry point for the shaper command."""
    # Check if binary exists
    if not BINARY_PATH.exists():
        print("Binary not found. Attempting to download...", file=sys.stderr)
        try:
            # Import and run install script
            from shaper_pkg import install as install_module
            install_module.main()
        except Exception as e:
            print(f"Failed to download binary: {e}", file=sys.stderr)
            print("Please run the install script manually or reinstall the package.", file=sys.stderr)
            sys.exit(1)
    
    # Replace current process with the Shaper binary to keep interactive
    # behavior identical to invoking the binary directly (no extra buffering).
    try:
        os.execv(str(BINARY_PATH), [str(BINARY_PATH), *sys.argv[1:]])
    except Exception as e:
        print(f"Error running shaper: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
