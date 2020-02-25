package mqtt

import (
	"flag"
	"fmt"
	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type messageProcessor func(topic, message string)

// SubscribeAll Subscribe all topic
func SubscribeAll(msgProc messageProcessor) {
	broker := flag.String("broker", "tcp://mosquitto:1883", "The broker URI. ex: tcp://localhost:1883")

	choke := make(chan MQTT.Message)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- msg
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := client.Subscribe("#", byte(0), nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	fmt.Println("Server started.")

	for {
		msg := <-choke
		topic, payload := msg.Topic(), (string(msg.Payload()))
		fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", topic, payload)
		msgProc(topic, payload)
	}
}
