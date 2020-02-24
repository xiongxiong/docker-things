* emqx [docker run -d --name emqx -p 1883:1883 -p 8083:8083 -p 8883:8883 -p 8084:8084 -p 18083:18083 emqx/emqx:v4.0.0]
* rabbitmq [docker run -d --name rabbitmq --hostname rabbitmq -p 5672:5672 rabbitmq:3]
* rabbitmq-admin [docker run -d --name rabbitmq-admin --hostname rabbitmq-admin -p 15672:15672 -e RABBITMQ_DEFAULT_USER=guest -e RABBITMQ_DEFAULT_PASS=guest rabbitmq:3-management]
* attach to container [docker exec -it mosquitto sh]
* restart linux docker service [systemctl restart docker]
* view port usage [sudo lsof -i -P -n | grep 5432] [sudo netstat -tanlp | grep 5432]
* view container logs [docker logs -f postgres]
* mosquitto publish a message [mosquitto_pub -t topic -m message]
* docker execute local script [docker exec mosquitto sh -c "`cat mqtt_pub.sh`"]