#!/usr/bin/env bash
set -o pipefail # abort on errors in pipeline
set -e          # abort on errors

# build netlog backup tool
echo "building netlog backup tool ..."
go build -o netlog-backup cmd/backups_cmd/main.go

echo "done <3"
