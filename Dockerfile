FROM golang:1.23.2 AS build

WORKDIR $GOPATH/src/hrtbt-dev.de/one-decrypt

# Install dependencies first, to enable caching for faster builds
COPY go.mod .
COPY go.sum .
# Fetch dependencies.
RUN go mod download
RUN go mod verify

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

COPY . .

# Build the binary
RUN go build -a -ldflags "-w -extldflags '-static'" -tags="no_duckdb_arrow" -o /usr/local/bin/shaper main.go

FROM scratch

COPY --from=build /usr/local/bin/shaper /usr/local/bin/shaper

CMD ["/usr/local/bin/shaper"]
