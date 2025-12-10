#!/usr/bin/env python3
# SPDX-License-Identifier: MPL-2.0

import os
import sys
import platform
import hashlib
import shutil
from pathlib import Path
import requests

# Get package directory
PACKAGE_DIR = Path(__file__).parent
BIN_DIR = PACKAGE_DIR / "bin"
VERSION_FILE = BIN_DIR / "VERSION"


def get_version():
    """Get the package version from distribution metadata."""
    try:
        import importlib.metadata
        return importlib.metadata.version("shaper-bin")
    except Exception:
        # Fallback: try to read from __init__.py or use default
        try:
            from shaper_pkg import __version__
            return __version__
        except Exception:
            return "0.0.0"  # Will be updated during build

# Map of platform names to GitHub release asset names
PLATFORM_MAP = {
    ("linux", "x86_64"): "shaper-linux-amd64",
    ("linux", "aarch64"): "shaper-linux-arm64",
    ("linux", "arm64"): "shaper-linux-arm64",
    ("darwin", "x86_64"): "shaper-darwin-amd64",
    ("darwin", "arm64"): "shaper-darwin-arm64",
    # Windows not yet supported
    # ("win32", "x86_64"): "shaper_Windows_x86_64.zip",
    # ("win32", "arm64"): "shaper_Windows_arm64.zip"
}


def get_release(version):
    """Fetch release information from GitHub API."""
    url = f"https://api.github.com/repos/taleshape-com/shaper/releases/tags/v{version}"
    headers = {
        "Accept": "application/vnd.github.v3+json",
        "User-Agent": "shaper-installer"
    }

    try:
        response = requests.get(url, headers=headers, timeout=30)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.HTTPError as e:
        if e.response.status_code == 404:
            raise Exception(f"Version v{version} not found. Please check if this version exists in the releases.")
        raise Exception(f"Error fetching release: {e}")
    except requests.exceptions.RequestException as e:
        raise Exception(f"Error fetching release: {e}")


def download_file(url, dest):
    """Download a file from URL to destination."""
    try:
        response = requests.get(url, stream=True, timeout=60)
        response.raise_for_status()

        # Check content length (max 100MB)
        content_length = response.headers.get("content-length")
        if content_length and int(content_length) > 200 * 1024 * 1024:
            raise Exception("File size exceeds 200MB limit")

        with open(dest, "wb") as f:
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)
    except requests.exceptions.RequestException as e:
        raise Exception(f"Error downloading file: {e}")


def calculate_sha256(file_path):
    """Calculate SHA256 checksum of a file."""
    sha256_hash = hashlib.sha256()
    with open(file_path, "rb") as f:
        for byte_block in iter(lambda: f.read(4096), b""):
            sha256_hash.update(byte_block)
    return sha256_hash.hexdigest()


def verify_checksum(file_path, expected_checksum):
    """Verify file checksum matches expected value."""
    actual_checksum = calculate_sha256(file_path)
    if actual_checksum != expected_checksum:
        raise Exception(
            f"Checksum verification failed for {Path(file_path).name}\n"
            f"Expected: {expected_checksum}\n"
            f"Actual:   {actual_checksum}"
        )


def load_checksums():
    """Load checksums from SHA256SUMS file."""
    checksums_path = BIN_DIR / "SHA256SUMS"
    if not checksums_path.exists():
        raise Exception("SHA256SUMS file not found in package")

    checksums = {}
    with open(checksums_path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                parts = line.split()
                if len(parts) >= 2:
                    checksum = parts[0]
                    filename = parts[1]
                    checksums[filename] = checksum

    return checksums


def main():
    """Main installation function."""
    try:
        # Create bin directory if it doesn't exist
        BIN_DIR.mkdir(parents=True, exist_ok=True)

        # Remove any stale marker so mismatched versions are re-fetched
        if VERSION_FILE.exists():
            VERSION_FILE.unlink()

        # Determine platform
        system = platform.system().lower()
        machine = platform.machine().lower()

        # Normalize machine names
        if machine == "amd64":
            machine = "x86_64"
        elif machine in ("arm64", "aarch64"):
            machine = "aarch64" if system == "linux" else "arm64"

        platform_key = (system, machine)
        asset_name = PLATFORM_MAP.get(platform_key)

        if not asset_name:
            # Try alternative mappings
            if system == "linux" and machine == "arm64":
                asset_name = PLATFORM_MAP.get(("linux", "aarch64"))
            elif system == "darwin" and machine == "aarch64":
                asset_name = PLATFORM_MAP.get(("darwin", "arm64"))

        if not asset_name:
            raise Exception(f"Unsupported platform: {system}-{machine}")

        # Load checksums from the package
        print("Loading checksums...")
        checksums = load_checksums()
        expected_checksum = checksums.get(asset_name)

        if not expected_checksum:
            raise Exception(f"No checksum found for {asset_name} in SHA256SUMS")

        # Get version
        version = get_version()

        # Get release information
        release = get_release(version)
        asset = next((a for a in release["assets"] if a["name"] == asset_name), None)

        if not asset:
            raise Exception(f"Asset not found for platform {system}-{machine} in version v{version}")

        # Download binary
        print(f"Downloading shaper v{version} for {system}-{machine}...")
        binary_path = BIN_DIR / "shaper"
        # Download into a temporary file to avoid leaving a partial binary
        tmp_path = BIN_DIR / ".shaper.tmp"
        download_file(asset["browser_download_url"], tmp_path)

        # Verify checksum
        print("Verifying checksum...")
        verify_checksum(tmp_path, expected_checksum)
        print("Checksum verified successfully")

        # Move into place and set executable permissions
        tmp_path.replace(binary_path)
        os.chmod(binary_path, 0o755)

        # Record the binary version shipped with this package
        VERSION_FILE.write_text(f"{version}\n", encoding="utf-8")

        print("Installation complete!")

    except Exception as error:
        print(f"Installation failed: {error}", file=sys.stderr)
        # Clean up binary if verification failed
        binary_path = BIN_DIR / "shaper"
        if binary_path.exists():
            binary_path.unlink()
        if (BIN_DIR / ".shaper.tmp").exists():
            (BIN_DIR / ".shaper.tmp").unlink()
        sys.exit(1)


if __name__ == "__main__":
    main()
