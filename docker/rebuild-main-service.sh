#!/bin/bash

#docker-compose -p serjservice stop -t 1 mainservice
#docker-compose -p serjservice build mainservice
#docker-compose -p serjservice up --no-start mainservice
#docker-compose -p serjservice start mainservice

# or better:
docker-compose -p serjservice up -d --force-recreate --no-deps --build mainservice
