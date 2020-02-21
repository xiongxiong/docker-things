package main

import (
	"flag"
	"fmt"
	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	SubscribeAll()
}

// SubscribeAll Subscribe all topic
func SubscribeAll() {
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
		PushToMQ(topic, payload)
	}
}

// PushToMQ push message to message queue
func PushToMQ(topic, message string) {

}

func Serv() {
	// r := gin.Default()
	// r.GET("/ping", func(c *gin.Context) {
	// 	c.JSON(200, gin.H{
	// 		"message": "pong",
	// 	})
	// })
	// r.GET("/sub/:broker/:topic", func(c *gin.Context) {
	// 	broker := c.Param("broker")
	// 	topic := c.Param("topic")
	// 	go mqtt.Sub(broker, topic)
	// 	c.JSON(200, gin.H{
	// 		"message": "ok",
	// 	})
	// })
	// r.Run()
}
