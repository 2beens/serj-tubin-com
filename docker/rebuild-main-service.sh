#!/bin/bash

docker-compose stop -t 1 goservice
docker-compose build goservice
docker-compose up --no-start goservice
docker-compose start goservice
