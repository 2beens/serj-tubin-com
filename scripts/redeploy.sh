#!/usr/bin/env bash
set -o pipefail # abort on errors in pipeline
set -e          # abort on errors

#   1 get branch name
skip_rebase=false
branch="master"
case $1 in
  "master")
    ;;
  "")
    ;;
  "-c")
    ;& # fallthru
  "--current")
    branch=$(git rev-parse --abbrev-ref HEAD)
    ;;
  "-cc")
    ;& # fallthru
  "--current-commit")
    skip_rebase=true
    ;;
  *)
    branch=$1
esac

if [[ "$skip_rebase" = false ]] ; then
  echo "--> deploying branch:" "${branch}"
  #   2 checkout branch
  git fetch --all
  git checkout ${branch}
  git rebase
else
  echo "--> skip fetch&rebase, just build and deploy the current commit"
fi

#   3 build project
echo "--> building project ..."
go build -o /home/serj/serj-tubin-com/service cmd/service/main.go
echo "--> build project done"

#   4 restart service and show info
echo "--> restarting service ..."
sudo systemctl restart serj-tubin-backend.service
sudo systemctl status serj-tubin-backend.service
echo "--> service restarted"

# build netlog backup tool (initiated by crontab)
echo "--> building netlog backup tool ..."
go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go

echo "==> all done! <3"
