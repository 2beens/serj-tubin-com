#!/bin/bash

if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
  echo "Usage: up.sh [--nerdctl|-nc for nerdctl instead of docker]"
  exit 0
fi

echo "Running docker compose for [$(uname -m)] ..."

use_nerdctl=false
if [ "$1" == "--nerdctl" ] || [ "$1" == "-nc" ]; then
  echo "Using nerdctl instead of docker"
  use_nerdctl=true
  shift
fi

docker() {
  if $use_nerdctl; then
    nerdctl "$@"
  else
    command docker "$@"
  fi
}

if [[ $(uname -m) == 'arm64' ]]; then
  docker compose -p serjservice \
    -f docker-compose.yml \
    -f docker-compose.apple-m1.yml \
    down
else
  docker compose -p serjservice down
fi
