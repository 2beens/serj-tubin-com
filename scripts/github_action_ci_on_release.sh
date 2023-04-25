#!/usr/bin/env bash

# Note: this script is being ran by GitHub Actions, upon new release. Check the repo actions for more details.

set -e # abort on errors

echo "running new release script ..."

cd /home/serj/serj-tubin-com
echo "current dir: $(pwd)"

#   1 get service
service="unknown"
case $1 in
  "mainservice")
    service="main"
    ;;
  "filebox")
    service="filebox"
    ;;
  *)
    branch=$1
esac

if [[ "$service" = "unknown" ]] ; then
  echo "unknown service: $1"
  exit 1
fi

#   2 checkout master branch
echo "--> git: checkout master ..."
git checkout master
echo "--> git: fetching ..."
git fetch --all
echo "--> git: rebase ..."
git rebase

#   3 build projects
if [[ "$service" = "main" ]] ; then
  echo "--> building main service ..."
  /usr/local/go/bin/go build -o /home/serj/serj-tubin-com/bin/service cmd/service/main.go
  echo "--> build main service done"
else
  echo "--> building file box service ..."
  /usr/local/go/bin/go build -o /home/serj/serj-tubin-com/bin/file-box cmd/file_service/main.go
  echo "--> build file box service done"
fi

#   4 restart services
if [[ "$service" = "main" ]] ; then
  echo "--> restarting main service ..."
  echo "${SERJ_PASS}\n" | sudo /bin/systemctl restart serj-tubin-backend.service
  echo "--> main service restarted"

  # build netlog backup tool (initiated by crontab)
  echo "--> building netlog backup tool ..."
  /usr/local/go/bin/go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go
else
  echo "--> restarting file box service ..."
  echo "${SERJ_PASS}\n" | sudo -S /bin/systemctl restart serj-tubin-file-box.service
  echo "--> file box service restarted"
fi

echo "==> all done! <3"
