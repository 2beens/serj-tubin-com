# Honeycomb OpenTelemetry Distro Changelog

## v0.3.0 (2022-10-31)

### Changes

- Minimum required Go version is 1.18 (#84)

### Maintenance

- Remove timestamp field from example trigger hook (#81) | @passcod
- Update launcher to latest (#80, #86) | @MikeGoldsmith @vreynolds
  - fix unconditional debug statements
  - update OTEL packages
- Bump go.opentelemetry.io/otel/exporters/stdout/stdouttrace from 1.9.0 to 1.11.1 (#84)
- Bump go.opentelemetry.io/otel/sdk from 1.9.0 to 1.10.0 (#76)

## v0.2.0 (2022-08-24)

### Enhancements

- Add local visualizations exporter (#66) | [@cartermp](https://github.com/cartermp)
- Add support for separate traces and metrics API key and dataset (#72) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)
- Disable metrics by default (#70) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)
- Add support for Honeycomb endpoint environment variables (#65) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)
- Add OTLP version header (#64) | [@vreynolds](https://github.com/vreynolds)

### Maintenance

- Add webhook triggers example (#68) | [@vreynolds](https://github.com/vreynolds)
- Add test matrix and nightly (#67) | [@vreynolds](https://github.com/vreynolds)

## v0.1.2 (2022-08-17)

## Fixed

- Set base exporter endpoint when setting up vendor opts (#56) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)
- Set log level to debug when debug option is set (#55) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)

### Maintenance

- Add baggage processor tests (#58) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)
- Add missing license headers (#57) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)
- More descriptive errors (#60) | [@cartermp](https://github.com/cartermp)

## v0.1.1 (2022-08-12)

### Fixed

- Update module path to match repo path (#46) | [@MikeGoldsmith](https://github.com/MikeGoldsmith)

### Maintenance

- Update README to clarify where most of the code lives (#45) | [@cartermp](https://github.com/cartermp)

## v0.1.0 (2022-08-12)

Initial release