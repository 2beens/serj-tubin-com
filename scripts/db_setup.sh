#!/bin/bash

set -e

if [ -z $DBNAME ]; then
    DBNAME=serj_blogs
fi

read -p "this will drop the current '$DBNAME' DB and recreate it, are you sure [y/n]? " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]
then
    exit
fi

echo "will re/create '$DBNAME' ..."

# drop all connections
psql -U postgres -h localhost -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname ='$DBNAME'"
# drop the database
dropdb -U postgres -h localhost --if-exists $DBNAME || echo "failed to drop db $DBNAME"
# create the database
createdb -U postgres -h localhost $DBNAME
# create db schema
psql -U postgres -h localhost -f ./sql/db_schema.sql $DBNAME
