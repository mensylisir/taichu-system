#!/bin/bash

export PGHOST=172.30.1.12
export PGPORT=32172
export PGUSER=postgres
export PGPASSWORD=Def@u1tpwd
export PGDATABASE=taichu

psql -f init-db.sql