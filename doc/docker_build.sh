#!/bin/bash
name_tag=wonderbear/thingspanel-doc:v1
docker build -t $name_tag . && docker push $name_tag