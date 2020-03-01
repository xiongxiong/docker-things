package mqtt

import (
	"dataservice/tool"
	"fmt"
	"log"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type messageProcessor func(topic, message string)

type broker struct {
	sync.RWMutex
	client   mqtt.Client
	mapTopic map[string]*struct{}
	chQuit   chan struct{}
	chMsg    chan mqtt.Message
}

type global struct {
	sync.RWMutex
	mapConn map[string]*broker
}

var _Global = global{
	mapConn: make(map[string]*broker),
}

// fetch or create it
func (_global *global) addBroker(brok string) *broker {
	_global.Lock()
	defer _global.Unlock()

	_broker := _global.mapConn[brok]
	if _broker == nil {
		chQuit := make(chan struct{})
		chMsg := make(chan mqtt.Message)

		opts := mqtt.NewClientOptions()
		opts.SetAutoReconnect(true)
		opts.AddBroker(brok)
		opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			chMsg <- msg
		})
		client := mqtt.NewClient(opts)

		_broker = &broker{
			client:   client,
			mapTopic: make(map[string]*struct{}),
			chQuit:   chQuit,
			chMsg:    chMsg,
		}
		_global.mapConn[brok] = _broker
	}

	return _broker
}

func (_global *global) delBroker(brok string) {
	_global.Lock()
	defer _global.Unlock()

	delete(_global.mapConn, brok)
}

func (_broker *broker) hasTopic(topic string) bool {
	_broker.RLock()
	defer _broker.RUnlock()

	return _broker.mapTopic[topic] != nil
}

func (_broker *broker) addTopic(topic string) {
	_broker.Lock()
	defer _broker.Unlock()

	if _broker.mapTopic[topic] == nil {
		_broker.mapTopic[topic] = &struct{}{}
	}
}

func (_broker *broker) delTopic(topic string) {
	_broker.Lock()
	defer _broker.Unlock()

	delete(_broker.mapTopic, topic)
	if len(_broker.mapTopic) == 0 {
		close(_broker.chQuit)
	}
}

// SubBrokerTopic Subscribe broker topic
func SubBrokerTopic(brok, topic string, msgProc messageProcessor) (err error) {
	return _Global.subBrokerTopic(brok, topic, msgProc)
}

func (_global *global) subBrokerTopic(brok, topic string, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_broker := _global.addBroker(brok)

	if _broker.client.IsConnected() == false {
		if token := _broker.client.Connect(); token.Wait() {
			tool.CheckThenPanic(token.Error(), "client connect")
		}
	}
	if _broker.hasTopic(topic) == false {
		if token := _broker.client.Subscribe(topic, byte(2), nil); token.Wait() {
			tool.CheckThenPanic(token.Error(), fmt.Sprintf("subscribe broker [%s] topic [%s]", brok, topic))
		}
		_broker.addTopic(topic)
	}

	go func() {
		quit := false
		for !quit {
			select {
			case msg := <-_broker.chMsg:
				topic, payload := msg.Topic(), (string(msg.Payload()))
				log.Printf("received topic: %s, message: %s\n", topic, payload)
				if msgProc != nil {
					go msgProc(topic, payload)
				}
			case <-_broker.chQuit:
				quit = true
				log.Printf("message channel for broker [%s] closed", brok)
			}
		}
		_broker.client.Disconnect(0)
		_global.delBroker(brok)
		log.Printf("connection for broker [%s] closed", brok)
	}()

	return
}

// UnSubBrokerTopic Unsubscribe broker topic
func UnSubBrokerTopic(brok, topic string) (err error) {
	return _Global.unSubBrokerTopic(brok, topic)
}

func (_global *global) unSubBrokerTopic(brok, topic string) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_broker := _global.mapConn[brok]
	if _broker == nil {
		panic("there is no such broker")
	}
	if _broker.hasTopic(topic) == false {
		panic("there is no such topic")
	}

	if token := _broker.client.Unsubscribe(topic); token.Wait() {
		tool.CheckThenPanic(token.Error(), fmt.Sprintf("unsubscribe broker [%s] topic [%s]", brok, topic))
	}
	_broker.delTopic(topic)
	return
}
