#!/bin/bash

docker-compose stop -t 1 mainservice
docker-compose build mainservice
docker-compose up --no-start mainservice
docker-compose start mainservice
