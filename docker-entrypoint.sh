#!/bin/sh
set -e

# If the first argument looks like a flag (starts with -), prepend the binary
# Also prepend the binary if it's a known subcommand
if [ "${1#-}" != "$1" ] || [ "$1" = 'dev' ] || [ "$1" = 'pull' ] || [ "$1" = 'deploy' ]; then
	set -- /usr/local/bin/shaper "$@"
fi

# If we are running the shaper binary and we are root,
# fix permissions on data directories and step down to shaper user
if [ "$1" = '/usr/local/bin/shaper' ] && [ "$(id -u)" = '0' ]; then
    # Ensure directories exist (in case they are not mounted)
    mkdir -p /data /var/lib/shaper

    # Fix permissions on data directories if they are owned by root.
    # We use a conditional check to avoid slow chown on every startup if already correct.
    # But since /data might have many files, we'll just do it if the root of the volume is root-owned.
    if [ "$(stat -c %u /data)" = '0' ]; then
        chown -R shaper:shaper /data
    fi
    if [ "$(stat -c %u /var/lib/shaper)" = '0' ]; then
        chown -R shaper:shaper /var/lib/shaper
    fi

    exec gosu shaper "$@"
fi

exec "$@"
