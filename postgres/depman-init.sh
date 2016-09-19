#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE USER depman PASSWORD 'depman';
    CREATE DATABASE depman;
    GRANT ALL PRIVILEGES ON DATABASE depman TO depman;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d depman -f /depman-schema.sql
