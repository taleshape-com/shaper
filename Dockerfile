FROM node:22-alpine AS frontend
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run lint
ENV NODE_ENV=production
RUN npm run build

FROM golang:1 AS build
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
RUN go build -a -ldflags "-w" -tags="no_duckdb_arrow" -o /usr/local/bin/shaper main.go

FROM debian:12-slim
# When running in a container, listen on all interfaces (including IPv6) by default
ENV SHAPER_ADDR=:3000
COPY --from=build /usr/local/bin/shaper /usr/local/bin/shaper
ENTRYPOINT ["/usr/local/bin/shaper"]
