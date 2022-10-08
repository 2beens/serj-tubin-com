#!/usr/bin/env bash
set -o pipefail # abort on errors in pipeline
set -e          # abort on errors

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd $SCRIPT_DIR/..

# build netlog backup tool
echo "building netlog backup tool ..."
go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go

echo "done <3"
