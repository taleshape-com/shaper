# Contributing

Thank you for wanting to help improve Shaper!

Please open an issue or start a discussion if you have any ideas or changes you like to contribute.
Contributions are very welcome, so don't hesitate to reach out even if you are unsure about it.
Let's just make sure we talk it through before writing code to make sure you spend your time on the right thing.


## Setup

1. Make sure you have [Node.js](https://nodejs.org/en/download/) and [Go](https://go.dev/doc/install) installed
2. To generate PDFs you also need Google Chrome or Chromium installed
3. Install dependencies: `npm install`


## Running Shaper from Source

1. Build the frontend: `npm run all`
2. Run the backend server: `go run .`
3. Access the app at [http://localhost:5454](http://localhost:5454)


## Developing Shaper

1. Run the backend server: `go run .`
2. Run the frontend: `npm start`
3. Access the app at [http://localhost:5453](http://localhost:5453)
4. Verify your changes:
  - `go test ./...`
  - `go vet ./...`
  - `npm run all`


## Releasing

Create a new Git tag to trigger a release.
