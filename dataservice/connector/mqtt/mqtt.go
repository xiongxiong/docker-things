package mqtt

import (
	"dataservice/tool"
	"fmt"
	"log"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type messageProcessor func(clientID string, msg mqtt.Message)

type client struct {
	username   string
	password   string
	mapBroker  map[string]struct{}
	mapTopic   map[string]byte
	mqttClient mqtt.Client
	chMsg      chan mqtt.Message
}

type global struct {
	sync.RWMutex
	mapClient map[string]*client
}

var _Global = global{
	mapClient: make(map[string]*client),
}

// get client
func (_global *global) getClient(clientID string) *client {
	_global.RLock()
	defer _global.RUnlock()

	return _global.mapClient[clientID]
}

// fetch or create it
func (_global *global) addClient(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte) *client {
	_client := client{
		username:  username,
		password:  password,
		mapBroker: mapBroker,
		mapTopic:  mapTopic,
		chMsg:     make(chan mqtt.Message),
	}

	opts := mqtt.NewClientOptions()
	opts.SetUsername(username)
	opts.SetPassword(password)
	for k := range _client.mapBroker {
		opts.AddBroker(k)
	}
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		_client.chMsg <- msg
	})
	_client.mqttClient = mqtt.NewClient(opts)

	_global.Lock()
	// delete old
	old := _global.mapClient[clientID]
	if old != nil {
		close(old.chMsg)
	}
	delete(_global.mapClient, clientID)

	// add new
	_global.mapClient[clientID] = &_client
	_global.Unlock()

	return &_client
}

func (_global *global) delClient(clientID string) {
	_global.Lock()
	defer _global.Unlock()

	_client := _global.mapClient[clientID]
	if _client != nil {
		close(_client.chMsg)
	}
	delete(_global.mapClient, clientID)
}

// Subscribe ...
func Subscribe(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte, msgProc messageProcessor) (err error) {
	return _Global.subscribe(clientID, username, password, mapBroker, mapTopic, msgProc)
}

func (_global *global) subscribe(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_client := _global.addClient(clientID, username, password, mapBroker, mapTopic)
	if token := _client.mqttClient.Connect(); token.Wait() {
		tool.CheckThenPanic(token.Error(), "client connect")
	}

	if token := _client.mqttClient.SubscribeMultiple(mapTopic, nil); token.Wait() {
		tool.CheckThenPanic(token.Error(), "client subscribe")
	}

	go func() {
		log.Printf("client [%s] listening ...", clientID)
		for {
			msg := <-_client.chMsg
			if msgProc != nil {
				go func() {
					defer func() {
						err = tool.Error(recover())
						tool.ErrorThenPrint(err, "message process")
					}()
					msgProc(clientID, msg)
				}()
			}
		}

		log.Printf("client [%s] disconnecting ...", clientID)
		var topics []string
		for k := range _client.mapTopic {
			topics = append(topics, k)
		}
		if token := _client.mqttClient.Unsubscribe(topics...); token.Wait() {
			tool.CheckThenPrint(token.Error(), fmt.Sprintf("unsubscribe client [%s]", clientID))
		}
		_client.mqttClient.Disconnect(0)
		log.Printf("client [%s] disconnected", clientID)
	}()

	return
}

// UnSubscribe ...
func UnSubscribe(clientID string) {
	_Global.unSubscribe(clientID)
}

func (_global *global) unSubscribe(clientID string) {
	_global.delClient(clientID)
}
