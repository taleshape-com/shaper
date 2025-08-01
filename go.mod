// SPDX-License-Identifier: MPL-2.0
module shaper

go 1.24

toolchain go1.24.2

require (
	github.com/golang-jwt/jwt/v5 v5.2.3
	github.com/jmoiron/sqlx v1.4.0
	github.com/labstack/echo-contrib v0.17.4
	github.com/labstack/echo-jwt/v4 v4.3.1
	github.com/labstack/echo/v4 v4.13.4
	github.com/labstack/gommon v0.4.2
	github.com/marcboeker/go-duckdb/v2 v2.3.3
	github.com/nats-io/nats-server/v2 v2.11.6
	github.com/nats-io/nats.go v1.43.0
	github.com/nrednav/cuid2 v1.1.0
	github.com/peterbourgon/ff/v4 v4.0.0-alpha.4
	github.com/samber/slog-echo v1.16.1
	github.com/stretchr/testify v1.10.0
	github.com/xuri/excelize/v2 v2.9.1
	golang.org/x/crypto v0.40.0
)

require (
	github.com/apache/arrow-go/v18 v18.4.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/duckdb/duckdb-go-bindings v0.1.17 // indirect
	github.com/duckdb/duckdb-go-bindings/darwin-amd64 v0.1.12 // indirect
	github.com/duckdb/duckdb-go-bindings/darwin-arm64 v0.1.12 // indirect
	github.com/duckdb/duckdb-go-bindings/linux-amd64 v0.1.12 // indirect
	github.com/duckdb/duckdb-go-bindings/linux-arm64 v0.1.12 // indirect
	github.com/duckdb/duckdb-go-bindings/windows-amd64 v0.1.12 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/flatbuffers v25.2.10+incompatible // indirect
	github.com/google/go-tpm v0.9.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/marcboeker/go-duckdb/arrowmapping v0.0.10 // indirect
	github.com/marcboeker/go-duckdb/mapping v0.0.11 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/highwayhash v1.0.3 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.7.4 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.22.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.65.0 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/richardlehane/mscfb v1.0.4 // indirect
	github.com/richardlehane/msoleps v1.0.4 // indirect
	github.com/samber/lo v1.51.0 // indirect
	github.com/tiendc/go-deepcopy v1.6.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/nfp v0.0.1 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792 // indirect
	golang.org/x/mod v0.26.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.35.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/marcboeker/go-duckdb/v2 v2.2.0 => github.com/taleshape-com/go-duckdb/v2 v2.2.1
