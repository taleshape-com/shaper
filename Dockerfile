FROM debian:12-slim

# When running in a container, listen on all interfaces (including IPv6) by default
ENV SHAPER_ADDR=:5454
# Override default data directory with something easy to mount to
ENV SHAPER_DIR=/data
# Override default DuckDB extension directory to persist downloaded extensions together with data
ENV SHAPER_DUCKDB_EXT_DIR=/data/duckdb_extensions

# Copy the correct binary based on architecture
COPY bin/shaper-${TARGETARCH} /usr/local/bin/shaper

ENTRYPOINT ["/usr/local/bin/shaper"]
