# SPDX-License-Identifier: MPL-2.0

# Using slim instead of a from scratch image because:
# 1. We are dynamically linking DuckDB since some extensions have issues with static linking
# 2. We need wget to run the healthcheck
# 3. Having a shell is useful for debugging
# Using Debian over Alpine since Debian uses glibc and DuckDB has issues with musl.
FROM debian:12.11-slim
LABEL maintainer="hi@taleshape.com"

ARG BUILD_DATE
ARG VERSION
LABEL org.opencontainers.image.authors="Taleshape OÜ"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.url="https://hub.docker.com/r/taleshape/shaper"
LABEL org.opencontainers.image.documentation="https://taleshape.com/shaper"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="Taleshape"
LABEL org.opencontainers.image.licenses="MPL-2.0"
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

# install wget for healthcheck
RUN apt-get update && apt-get install -y wget && rm -rf /var/lib/apt/lists/*

# Copy the correct binary based on architecture
COPY bin/shaper-linux-${TARGETARCH} /usr/local/bin/shaper

ENTRYPOINT ["/usr/local/bin/shaper"]
