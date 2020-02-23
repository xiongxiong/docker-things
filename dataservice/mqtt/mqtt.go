package mqtt

import (
	"flag"
	"fmt"
	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// SubscribeAll Subscribe all topic
func SubscribeAll(msgProc func(topic, message string)) {
	broker := flag.String("broker", "tcp://localhost:1883", "The broker URI. ex: tcp://localhost:1883")

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

// Sub ...
func Sub(broker string, topic string) {
	choke := make(chan [2]string)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := client.Subscribe(topic, byte(0), nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	for {
		incoming := <-choke
		fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", incoming[0], incoming[1])
	}

	client.Disconnect(250)
	fmt.Println("Sample Subscriber Disconnected")
}
