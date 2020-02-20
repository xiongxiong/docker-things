package mqtt

import (
	"fmt"
	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func Sub() {
	broker := "tcp://localhost:1883"
	topic := "hello"

	receiveCount := 0
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
	for receiveCount < 3 {
		incoming := <-choke
		fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", incoming[0], incoming[1])
		receiveCount++
	}

	client.Disconnect(250)
	fmt.Println("Sample Subscriber Disconnected")
}
