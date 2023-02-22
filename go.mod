module github.com/2beens/serjtubincom

go 1.18

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/aerospike/aerospike-client-go v3.0.0+incompatible
	github.com/brianvoe/gofakeit/v6 v6.19.0
	github.com/coocood/freecache v1.1.0
	github.com/dgraph-io/ristretto v0.0.3
	github.com/exaring/otelpgx v0.3.1
	github.com/go-redis/redis/extra/redisotel/v8 v8.11.5
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-redis/redis_rate/v9 v9.1.2
	github.com/go-redis/redismock/v8 v8.0.6
	github.com/gorilla/mux v1.8.0
	github.com/honeycombio/honeycomb-opentelemetry-go v0.3.0
	github.com/honeycombio/opentelemetry-go-contrib/launcher v0.0.0-20221031150637-a3c60ed98d54
	github.com/ipinfo/go/v2 v2.9.1
	github.com/jackc/pgx/v5 v5.1.1
	github.com/prometheus/client_golang v1.14.0
	github.com/prometheus/client_model v0.3.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.8.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux v0.36.4
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.36.4
	go.opentelemetry.io/otel v1.11.1
	go.uber.org/goleak v1.2.0
	go.uber.org/multierr v1.8.0
	golang.org/x/crypto v0.1.0
	google.golang.org/api v0.100.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

require (
	cloud.google.com/go/compute v1.10.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/felixge/httpsnoop v1.0.3 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-redis/redis/extra/rediscmd/v8 v8.11.5 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.0 // indirect
	github.com/googleapis/gax-go/v2 v2.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.12.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/puddle/v2 v2.1.2 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20220913051719-115f729f3c8c // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20220216144756-c35f1ee13d7c // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/rogpeppe/go-internal v1.6.1 // indirect
	github.com/sethvargo/go-envconfig v0.8.2 // indirect
	github.com/shirou/gopsutil/v3 v3.22.9 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.5.0 // indirect
	github.com/yuin/gopher-lua v0.0.0-20200603152657-dc2b0ca8b37e // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/host v0.36.4 // indirect
	go.opentelemetry.io/contrib/instrumentation/runtime v0.36.4 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.11.1 // indirect
	go.opentelemetry.io/contrib/propagators/ot v1.11.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.11.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v0.33.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.33.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v0.33.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.11.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.11.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.11.1 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.11.1 // indirect
	go.opentelemetry.io/otel/metric v0.33.0 // indirect
	go.opentelemetry.io/otel/sdk v1.11.1 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.33.0 // indirect
	go.opentelemetry.io/otel/trace v1.11.1 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/oauth2 v0.0.0-20221014153046-6fdb5e3db783 // indirect
	golang.org/x/sync v0.0.0-20220929204114-8fcdb60fdcc0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c // indirect
	google.golang.org/grpc v1.50.1 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
