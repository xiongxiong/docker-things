mosquitto_pub -t "topic" -m "kkk"

docker exec mosquitto sh -c "`cat mqtt_pub.sh`"