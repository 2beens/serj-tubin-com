#!/usr/bin/env bash

# Note: this script is being ran by GitHub Actions, upon new release. Check the repo actions for more details.

set -e # abort on errors

echo "running new release script ..."

cd /home/serj/serj-tubin-com
echo "current dir: $(pwd)"

echo "--> git: fetching ..."
git fetch --all
echo "--> git: checkout ..."
git checkout ci-release
echo "--> git: rebase ..."
git rebase

#   3 build project
echo "--> building project ..."
/usr/local/go/bin/go build -o /home/serj/serj-tubin-com/bin/service cmd/service/main.go
echo "--> build project done"

#   4 restart service and show info
echo "--> restarting service ..."
echo "${SERJ_PASS}\n" | sudo /bin/systemctl restart serj-tubin-backend.service
echo "--> service restarted"

# build netlog backup tool (initiated by crontab)
echo "--> building netlog backup tool ..."
/usr/local/go/bin/go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go

echo "==> all done! <3"
