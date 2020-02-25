#!/bin/bash
name_tag=wonderbear/thingspanel-dataservice:v1
go build . && docker build -t $name_tag . && docker push $name_tag