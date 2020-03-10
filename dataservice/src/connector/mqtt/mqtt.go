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

// Manager ...
type Manager struct {
	lock      sync.RWMutex
	mapClient map[string]*client
}

// NewManager ...
func NewManager() *Manager {
	return &Manager{
		mapClient: make(map[string]*client),
	}
}

// get client
func (_manager *Manager) getClient(clientID string) *client {
	_manager.lock.RLock()
	defer _manager.lock.RUnlock()

	return _manager.mapClient[clientID]
}

// fetch or create it
func (_manager *Manager) addClient(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte) *client {
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

	_manager.lock.Lock()
	// delete old
	old := _manager.mapClient[clientID]
	if old != nil {
		close(old.chMsg)
	}
	delete(_manager.mapClient, clientID)

	// add new
	_manager.mapClient[clientID] = &_client
	_manager.lock.Unlock()

	return &_client
}

func (_manager *Manager) delClient(clientID string) {
	_manager.lock.Lock()
	defer _manager.lock.Unlock()

	_client := _manager.mapClient[clientID]
	if _client != nil {
		close(_client.chMsg)
	}
	delete(_manager.mapClient, clientID)
}

// Subscribe ...
func (_manager *Manager) Subscribe(clientID, username, password string, mapBroker map[string]struct{}, mapTopic map[string]byte, msgProc messageProcessor) (err error) {
	defer func() {
		err = tool.Error(recover())
	}()

	_client := _manager.addClient(clientID, username, password, mapBroker, mapTopic)
	if token := _client.mqttClient.Connect(); token.Wait() {
		tool.CheckThenPanic(token.Error(), "client connect")
	}

	if token := _client.mqttClient.SubscribeMultiple(mapTopic, nil); token.Wait() {
		tool.CheckThenPanic(token.Error(), "client subscribe")
	}

	go func() {
		log.Printf("client [%s] listening ...", clientID)
		for {
			msg, ok := <-_client.chMsg
			if !ok {
				break
			}
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
func (_manager *Manager) UnSubscribe(clientID string) {
	_manager.delClient(clientID)
}
