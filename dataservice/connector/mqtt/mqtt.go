package mqtt

import (
	"dataservice/tool"
	"fmt"
	"log"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type messageProcessor func(topic, message string)

type clientK struct {
	broker   string
	username string
}

type clientV struct {
	sync.RWMutex
	mapTopic map[string]*struct{}
	client   mqtt.Client
	chQuit   chan struct{}
	chMsg    chan mqtt.Message
}

type global struct {
	sync.RWMutex
	mapConn map[clientK]*clientV
}

var _Global = global{
	mapConn: make(map[clientK]*clientV),
}

// get client
func (_global *global) getClient(user, pass, brok string) *clientV {
	_global.Lock()
	defer _global.Unlock()

	_clientK := clientK{
		broker:   brok,
		username: user,
	}
	return _global.mapConn[_clientK]
}

// fetch or create it
func (_global *global) addClient(user, pass, brok string) (err error) {
	_global.Lock()
	defer _global.Unlock()

	_clientK := clientK{
		broker:   brok,
		username: user,
	}
	_clientV := _global.mapConn[_clientK]
	if _clientV == nil {
		chQuit := make(chan struct{})
		chMsg := make(chan mqtt.Message)

		opts := mqtt.NewClientOptions()
		opts.SetAutoReconnect(true)
		opts.SetUsername(user)
		opts.SetPassword(pass)
		opts.AddBroker(brok)
		opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
			chMsg <- msg
		})
		client := mqtt.NewClient(opts)

		if token := client.Connect(); token.Wait() {
			if token.Error() != nil {
				err = token.Error()
				return
			}
		}

		_clientV = &clientV{
			client:   client,
			mapTopic: make(map[string]*struct{}),
			chQuit:   chQuit,
			chMsg:    chMsg,
		}
		_global.mapConn[_clientK] = _clientV
	}

	return
}

func (_global *global) delClient(brok string) {
	_global.Lock()
	defer _global.Unlock()

	delete(_global.mapConn, brok)
}

func (_clientV *clientV) hasTopic(topic string) bool {
	_clientV.RLock()
	defer _clientV.RUnlock()

	return _clientV.mapTopic[topic] != nil
}

func (_clientV *clientV) addTopic(topic string) {
	_clientV.Lock()
	defer _clientV.Unlock()

	if _clientV.mapTopic[topic] == nil {
		_clientV.mapTopic[topic] = &struct{}{}
	}
}

func (_clientV *clientV) delTopic(topic string) {
	_clientV.Lock()
	defer _clientV.Unlock()

	delete(_clientV.mapTopic, topic)
	if len(_clientV.mapTopic) == 0 {
		close(_clientV.chQuit)
	}
}

// SubClientTopic Subscribe broker topic
func SubClientTopic(user, pass, brok, topic string, msgProc messageProcessor) (err error) {
	return _Global.subClientTopic(user, pass, brok, topic, msgProc)
}

func (_global *global) subClientTopic(user, pass, brok, topic string, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_clientV := _global.addClient(user, pass, brok)

	if _clientV.hasTopic(topic) == false {
		if token := _clientV.client.Subscribe(topic, byte(2), nil); token.Wait() {
			tool.CheckThenPanic(token.Error(), fmt.Sprintf("subscribe broker [%s] topic [%s]", brok, topic))
		}
		_clientV.addTopic(topic)
	}

	go func() {
		quit := false
		for !quit {
			select {
			case msg := <-_clientV.chMsg:
				topic, payload := msg.Topic(), (string(msg.Payload()))
				log.Printf("received topic: %s, message: %s\n", topic, payload)
				if msgProc != nil {
					go msgProc(topic, payload)
				}
			case <-_clientV.chQuit:
				quit = true
				log.Printf("message channel for broker [%s] closed", brok)
			}
		}
		_clientV.client.Disconnect(0)
		_global.delClient(brok)
		log.Printf("connection for broker [%s] closed", brok)
	}()

	return
}

// UnSubClientTopic Unsubscribe broker topic
func UnSubClientTopic(brok, topic string) (err error) {
	return _Global.unSubClientTopic(brok, topic)
}

func (_global *global) unSubClientTopic(brok, topic string) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_clientV := _global.mapConn[brok]
	if _clientV == nil {
		panic("there is no such broker")
	}
	if _clientV.hasTopic(topic) == false {
		panic("there is no such topic")
	}

	if token := _clientV.client.Unsubscribe(topic); token.Wait() {
		tool.CheckThenPanic(token.Error(), fmt.Sprintf("unsubscribe broker [%s] topic [%s]", brok, topic))
	}
	_clientV.delTopic(topic)
	return
}
