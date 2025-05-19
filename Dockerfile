FROM debian:12-slim
LABEL maintainer="Taleshape <hi@taleshape.com>"
LABEL org.opencontainers.image.source="https://github.com/taleshape-com/shaper"

ARG TARGETARCH

# When running in a container, listen on all interfaces (including IPv6) by default
ENV SHAPER_ADDR=:5454
# Override default data directory with something easy to mount to
ENV SHAPER_DIR=/data
# Override default DuckDB extension directory to persist downloaded extensions together with data
ENV SHAPER_DUCKDB_EXT_DIR=/data/duckdb_extensions
ENV SHAPER_INIT_SQL_FILE=/var/lib/shaper/init.sql

EXPOSE 5454
HEALTHCHECK CMD ["wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:5454/health"]

RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

# Copy the correct binary based on architecture
COPY bin/shaper-linux-${TARGETARCH} /usr/local/bin/shaper

ENTRYPOINT ["/usr/local/bin/shaper"]
