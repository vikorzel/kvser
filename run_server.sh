#!/bin/bash
docker-compose pull
docker-compose up -d
docker-compose exec tarantool tarantool /opt/tarantool/lua/app.lua
