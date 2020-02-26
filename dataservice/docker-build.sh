#!/bin/bash
name_tag=wonderbear/thingspanel-dataservice:v1
GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o bin/dataservice . && docker build -t $name_tag .
# GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o bin/dataservice . && docker build -t $name_tag . && docker push $name_tag