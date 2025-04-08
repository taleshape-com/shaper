FROM golang:1 AS build
WORKDIR $GOPATH/src/shaper
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go mod verify
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}
ENV CGO_ENABLED=1
COPY . .
# The dist directory with frontend assets is already in the build context
RUN go build -a -ldflags "-w" -tags="no_duckdb_arrow" -o /usr/local/bin/shaper main.go

FROM debian:12-slim
# When running in a container, listen on all interfaces (including IPv6) by default
ENV SHAPER_ADDR=:5454
# Override default data directory with something easy to mount to
ENV SHAPER_DIR=/data
# Override default DuckDB extension directory to persist downloaded extensions together with data
# Helps avoid downloading extensions every time the container starts
ENV SHAPER_DUCKDB_EXT_DIR=/data/duckdb_extensions
COPY --from=build /usr/local/bin/shaper /usr/local/bin/shaper
ENTRYPOINT ["/usr/local/bin/shaper"]
