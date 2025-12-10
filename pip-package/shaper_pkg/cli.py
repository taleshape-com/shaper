#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0

"""Command-line interface for Shaper."""

import os
import sys
from pathlib import Path

# Get package directory
PACKAGE_DIR = Path(__file__).parent
BINARY_PATH = PACKAGE_DIR / "bin" / "shaper"
VERSION_PATH = PACKAGE_DIR / "bin" / "VERSION"


def _read_binary_version():
    """Return the version recorded for the installed binary, if any."""
    try:
        return VERSION_PATH.read_text(encoding="utf-8").strip()
    except OSError:
        return None


def _ensure_binary():
    """Download the binary if missing or out-of-date."""
    # Deferred import so we don't pay the cost when just exec'ing
    from shaper_pkg import install as install_module

    desired_version = install_module.get_version()
    current_version = _read_binary_version()

    needs_download = (not BINARY_PATH.exists()) or (current_version != desired_version)

    if needs_download:
        print("Fetching Shaper binary...", file=sys.stderr)
        install_module.main()


def main():
    """Main entry point for the shaper command."""
    try:
        _ensure_binary()
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
