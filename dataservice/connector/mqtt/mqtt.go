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

type clients struct {
	sync.RWMutex
	mapClient map[string]*client
}

var _Clients = clients{
	mapClient: make(map[string]*client),
}

// get client
func (_clients *clients) getClient(clientID string) *client {
	_clients.RLock()
	defer _clients.RUnlock()

	return _clients.mapClient[clientID]
}

// fetch or create it
func (_clients *clients) addClient(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte) *client {
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

	_clients.Lock()
	// delete old
	old := _clients.mapClient[clientID]
	if old != nil {
		close(old.chMsg)
	}
	delete(_clients.mapClient, clientID)

	// add new
	_clients.mapClient[clientID] = &_client
	_clients.Unlock()

	return &_client
}

func (_clients *clients) delClient(clientID string) {
	_clients.Lock()
	defer _clients.Unlock()

	_client := _clients.mapClient[clientID]
	if _client != nil {
		close(_client.chMsg)
	}
	delete(_clients.mapClient, clientID)
}

// Subscribe ...
func Subscribe(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte, msgProc messageProcessor) (err error) {
	return _Clients.subscribe(clientID, username, password, mapBroker, mapTopic, msgProc)
}

func (_clients *clients) subscribe(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_client := _clients.addClient(clientID, username, password, mapBroker, mapTopic)
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
	_Clients.unSubscribe(clientID)
}

func (_clients *clients) unSubscribe(clientID string) {
	_clients.delClient(clientID)
}
