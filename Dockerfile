FROM node:22.11.0 AS frontend
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install
ENV NODE_ENV=production
COPY . .
RUN npm run build

FROM golang:1.23.2 AS build
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
RUN go build -a -ldflags "-w -extldflags '-static'" -tags="no_duckdb_arrow" -o /usr/local/bin/shaper main.go

FROM scratch
COPY --from=build /usr/local/bin/shaper /usr/local/bin/shaper
ENTRYPOINT ["/usr/local/bin/shaper"]
