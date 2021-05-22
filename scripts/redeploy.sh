#!/usr/bin/env bash
set -o pipefail # abort on errors in pipeline
set -e          # abort on errors

# TODO: maybe read the branch name from stdin
# read -p "branch name: " branch

#   1 get branch name
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
    *)
        branch=$1
esac

echo "--> deploying branch:" "${branch}"

#   2 checkout branch
git fetch --all
git checkout ${branch}
git rebase

#   3 build project
echo "--> building project ..."
go build -o /home/serj/serj-tubin-com/service cmd/service/main.go
echo "--> build project done"

#   4 restart service and show info
echo "--> restarting service ..."
sudo systemctl restart serj-tubin-backend.service
sudo systemctl status serj-tubin-backend.service
echo "--> service restarted"

# build netlog backup tool
echo "--> building netlog backup tool ..."
go build -o /home/serj/serj-tubin-com/netlog-backup cmd/netlog_gd_backup/main.go

echo "==> all done! <3"
