for i in $(seq 1 5)
do
mosquitto_pub -t "topic" -m "message $i"
done