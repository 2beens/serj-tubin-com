#!/usr/bin/env bash
set -o pipefail # abort on errors in pipeline
set -e          # abort on errors

# make sure aerospike is running within Vagrant

go run cmd/service/main.go -port=9000 -loglvl=trace -ahost=172.28.128.3 -aero-namespace=test -logs-path=
