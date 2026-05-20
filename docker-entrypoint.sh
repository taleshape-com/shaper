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

    # Fix permissions on data directories and files if they are owned by root.
    # We do this even if the directory itself isn't root-owned, to catch bind-mounted files.
    # Using find to only chown root-owned items to keep it relatively fast.
    find /data /var/lib/shaper -maxdepth 2 -user root -exec chown -R shaper:shaper {} +

    exec gosu shaper "$@"
fi

exec "$@"
