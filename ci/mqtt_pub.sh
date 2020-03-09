for i in $(seq 1 5)
do
mosquitto_pub -t "topic" -m "{\"temperature\": $(($RANDOM % 100)), \"time\": $(date +%s)}"
done