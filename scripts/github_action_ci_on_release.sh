#!/usr/bin/env bash

# Note: this script is being ran by GitHub Actions, upon new release. Check the repo actions for more details.

set -e # abort on errors

echo "running new release script ..."

cd /home/serj/serj-tubin-com
echo "current dir: $(pwd)"

echo "--> git: checkout master ..."
git checkout master
echo "--> git: fetching ..."
git fetch --all
echo "--> git: rebase ..."
git rebase

#   3 build projects
echo "--> building main service ..."
/usr/local/go/bin/go build -o /home/serj/serj-tubin-com/bin/service cmd/service/main.go
echo "--> build main service done"
echo "--> building file box service ..."
/usr/local/go/bin/go build -o /home/serj/serj-tubin-com/bin/file-box cmd/file_service/main.go
echo "--> build file box service done"

#   4 restart services
echo "--> restarting main service ..."
echo "${SERJ_PASS}\n" | sudo /bin/systemctl restart serj-tubin-backend.service
echo "--> main service restarted"
echo "--> restarting file box service ..."
echo "${SERJ_PASS}\n" | sudo /bin/systemctl restart serj-tubin-file-box.service
echo "--> file box service restarted"

# build netlog backup tool (initiated by crontab)
echo "--> building netlog backup tool ..."
/usr/local/go/bin/go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go

echo "==> all done! <3"
