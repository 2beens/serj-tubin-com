#!/usr/bin/env bash
set -o pipefail # abort on errors in pipeline
set -e          # abort on errors

# build netlog backup tool
echo "building netlog backup tool ..."
go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go

echo "done <3"
