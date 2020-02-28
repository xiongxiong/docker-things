package mqtt

import (
	"dataservice/tool"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type messageProcessor func(topic, message string)

type conn struct {
	client mqtt.Client
	chQuit chan struct{}
	chMsg  chan mqtt.Message
}

type global struct {
	mapConn map[string]*conn
}

var _global = global{
	mapConn: make(map[string]*conn),
}

// SubBrokerTopic Subscribe broker topic
func SubBrokerTopic(broker, topic string, msgProc messageProcessor) (err error) {
	return _global.subBrokerTopic(broker, topic, msgProc)
}

func (glob *global) subBrokerTopic(broker, topic string, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	var client mqtt.Client
	var chQuit chan struct{}
	var chMsg chan mqtt.Message

	if _conn := glob.mapConn[broker]; _conn == nil {
		chQuit = make(chan struct{})
		chMsg = make(chan mqtt.Message)

		opts := mqtt.NewClientOptions()
		opts.SetAutoReconnect(true)
		opts.AddBroker(broker)
		opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			chMsg <- msg
		})
		client = mqtt.NewClient(opts)

		glob.mapConn[broker] = &conn{client, chQuit, chMsg}
	} else {
		client = _conn.client
		chQuit = _conn.chQuit
		chMsg = _conn.chMsg
	}

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := client.Subscribe(topic, byte(2), nil); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	log.Printf("subscribe broker success -- %s", broker)

	go func() {
		quit := false
		for !quit {
			select {
			case msg := <-chMsg:
				topic, payload := msg.Topic(), (string(msg.Payload()))
				log.Printf("received topic: %s, message: %s\n", topic, payload)
				msgProc(topic, payload)
			case <-chQuit:
				quit = true
				log.Printf("message channel for broker [%s] closed", broker)
			}
		}
		client.Disconnect(1000)
		log.Printf("connection for broker [%s] closed", broker)
	}()

	return
}

// UnSubBrokerTopic Unsubscribe broker topic
func UnSubBrokerTopic(broker, topic string) (err error) {
	return _global.unSubBrokerTopic(broker, topic)
}

func (glob *global) unSubBrokerTopic(broker, topic string) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	if _conn := glob.mapConn[broker]; _conn != nil {

	}
	return
}
