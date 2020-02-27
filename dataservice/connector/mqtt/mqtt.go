package mqtt

import (
	"dataservice/tool"
	"log"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type messageProcessor func(topic, message string)

// SubBrokerTopic Subscribe broker topic
func SubBrokerTopic(broker, topic string, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	choke := make(chan MQTT.Message)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		choke <- msg
	})

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := client.Subscribe(topic, byte(0), nil); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Printf("subscribe broker success -- %s", broker)

	go func() {
		for {
			msg := <-choke
			topic, payload := msg.Topic(), (string(msg.Payload()))
			log.Printf("Received topic: %s, message: %s\n", topic, payload)
			msgProc(topic, payload)
		}
	}()

	return
}
