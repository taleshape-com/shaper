FROM node:22.11.0 AS frontend
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run lint
ENV NODE_ENV=production
RUN npm run build

FROM golang:1.23.2 AS build
WORKDIR $GOPATH/src/shaper

# Install build dependencies for DuckDB postgres extension
RUN apt-get update && apt-get install -y \
    git \
    build-essential \
    cmake \
    libpq-dev

# Clone and build the postgres extension
WORKDIR /tmp/duckdb-postgres
RUN git clone --recursive https://github.com/duckdb/duckdb-postgres.git . && \
    make

# Create DuckDB extension directory structure
RUN mkdir -p /duckdb/extensions/v1.1.3/linux_amd64

# Copy the built extension to the DuckDB extensions directory
RUN cp build/release/extension/postgres_scanner/postgres_scanner.duckdb_extension \
    /duckdb/extensions/v1.1.3/linux_amd64/

# Back to building the main application
WORKDIR $GOPATH/src/shaper
COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go mod verify
ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64
COPY . .
COPY --from=frontend /app/dist dist
RUN go vet ./...
RUN go build -a -ldflags "-w -extldflags '-static'" -tags="no_duckdb_arrow" -o /usr/local/bin/shaper main.go

FROM debian:bullseye-slim
# Install required runtime dependencies
RUN apt-get update && apt-get install -y \
    libpq5 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /usr/local/bin/shaper /usr/local/bin/shaper
COPY --from=build /duckdb/extensions /root/.duckdb/extensions

ENTRYPOINT ["/usr/local/bin/shaper"]
