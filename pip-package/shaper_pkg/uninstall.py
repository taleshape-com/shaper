#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0

import sys
from pathlib import Path

# Get package directory
PACKAGE_DIR = Path(__file__).parent
BIN_DIR = PACKAGE_DIR / "bin"
BINARY_PATH = BIN_DIR / "shaper"

try:
    if BINARY_PATH.exists():
        BINARY_PATH.unlink()
    if BIN_DIR.exists() and not any(BIN_DIR.iterdir()):
        BIN_DIR.rmdir()
except Exception as error:
    print(f"Error during uninstall: {error}", file=sys.stderr)
    sys.exit(1)
