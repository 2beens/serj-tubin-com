#!/bin/bash

docker-compose stop -t 1 fileservice
docker-compose build fileservice
docker-compose up --no-start fileservice
docker-compose start fileservice
