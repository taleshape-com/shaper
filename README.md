# Shaper

**Shaper is a minimal data platform built on top of DuckDB and NATS to create analytics dashboards and embed them into your software.**

Learn more at: https://taleshape.com/shaper/docs/

[
![Screenshot](https://taleshape.com/_astro/session_dashboard.DjtFqCnO_Z15ug1D.webp)
](https://taleshape.com/shaper/docs/)


## Quickstart

The quickest way to try out Shaper without installing anything is to run it via [Docker](https://www.docker.com/):
```sh
docker run --rm -it -p5454:5454 taleshape/shaper
```
Then open http://localhost:5454/new in your browser.

For more, checkout the [Getting Started Guide](https://taleshape.com/shaper/docs/getting-started/).

And to run Shaper in production, see the [Deployment Guide](https://taleshape.com/shaper/docs/deploy-to-production/).


## Get in Touch

Feel free to open an [issue](https://github.com/taleshape-com/shaper/issues) or start a [discussion](https://github.com/taleshape-com/shaper/discussions) if you have any questions or suggestions.

Also follow along on [BlueSky](https://bsky.app/profile/taleshape.bsky.social) or [LinkedIn](https://www.linkedin.com/company/taleshape/).

And subscribe to our [newsletter](https://taleshape.com/newsletter) to get updates about Shaper.


## Development Setup

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


### Releasing

Create a new Git tag to trigger a release.


## Release Notes

See [Github Releases](https://github.com/taleshape-com/shaper/releases)


## License and Copyright

Shaper is licensed under the [Mozilla Public License 2.0](https://github.com/taleshape-com/shaper/blob/main/LICENSE).

Copyright © 2024-2025 Taleshape OÜ
