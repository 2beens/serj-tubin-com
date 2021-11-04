#!/bin/bash

docker exec -it st-redis redis-cli -a ${SERJ_REDIS_PASS}
