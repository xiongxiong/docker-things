#!/bin/bash

docker run -p 9000:9000 -p 35729:35729 --rm -d --name doc wonderbear/thingspanel-doc:v1
# docker run -v "$PWD":/data/dist/apis -p 9000:9000 -p 35729:35729 --rm -d --name doc slecache/api-console