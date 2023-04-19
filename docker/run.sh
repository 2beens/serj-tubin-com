#!/bin/bash

if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
  print_usage_and_exit
fi

num_args=$#
if [ $num_args -lt 1 ]; then
  print_usage_and_exit
fi

if [ "$1" != "up" ] && [ "$1" != "down" ]; then
  echo "Error: first argument must be either 'up' or 'down'"
  print_usage_and_exit
fi

use_nerdctl=false
if [ "$2" == "--nerdctl" ] || [ "$2" == "-nc" ]; then
  echo "Using nerdctl instead of docker"
  use_nerdctl=true
fi

docker() {
  if $use_nerdctl; then
    nerdctl "$@"
  else
    command docker "$@"
  fi
}

echo "Running docker compose [$1] for [$(uname -m)], use nerdctl [$use_nerdctl] ..."

if [ "$1" == "up" ]; then
  if [[ $(uname -m) == 'arm64' ]]; then
    echo "Using up on docker-compose.apple-m1.yml additionally ..."
    docker compose -p serjservice \
      -f docker-compose.yml \
      -f docker-compose.apple-m1.yml \
      up --build -d
  else
    docker compose -p serjservice up --build -d
  fi
else # down
  if [[ $(uname -m) == 'arm64' ]]; then
    echo "Using down on docker-compose.apple-m1.yml additionally ..."
    docker compose -p serjservice \
      -f docker-compose.yml \
      -f docker-compose.apple-m1.yml \
      down
  else
    docker compose -p serjservice down
  fi
fi

echo "Done."

############ FUNCTIONS ###################
print_usage_and_exit() {
  echo "Usage: run.sh [up|down] [--nerdctl|-nc for nerdctl instead of docker]"
  exit 0
}
