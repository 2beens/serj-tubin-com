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
echo "--> building file box service ..."
go build -o /home/serj/serj-tubin-com/bin/file-box cmd/file_service/main.go
echo "--> build file box service done"

#   4 restart service and show info
echo "--> restarting file box service ..."
sudo systemctl restart serj-tubin-file-box.service
sudo systemctl status serj-tubin-file-box.service
echo "--> file box service restarted"
echo "==> all done! <3"
