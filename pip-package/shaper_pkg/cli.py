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
    if not BINARY_PATH.exists():
        print(f"Error: Shaper binary not found at {BINARY_PATH}", file=sys.stderr)
        print("This can happen if you installed from a source distribution on an unsupported platform.", file=sys.stderr)
        print("Please download the binary for your platform from:", file=sys.stderr)
        print("https://github.com/taleshape-com/shaper/releases", file=sys.stderr)
        sys.exit(1)

    # Ensure the binary is executable
    if not os.access(BINARY_PATH, os.X_OK):
        try:
            os.chmod(BINARY_PATH, 0o755)
        except Exception as e:
            print(f"Warning: Failed to set executable permissions on {BINARY_PATH}: {e}", file=sys.stderr)
    
    # Replace current process with the Shaper binary to keep interactive
    # behavior identical to invoking the binary directly (no extra buffering).
    try:
        os.execv(str(BINARY_PATH), [str(BINARY_PATH), *sys.argv[1:]])
    except Exception as e:
        print(f"Error running shaper: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
