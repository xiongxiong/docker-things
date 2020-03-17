package controller

import (
	"context"
	"database/sql"
	mqttC "dataservice/connector/mqtt"
	"dataservice/store"
	"dataservice/tool"
	"encoding/json"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	gin "github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

type topic struct {
	Topic string `json:"topic"`
	Qos   byte   `json:"qos"`
}

type client struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Brokers  []string `json:"brokers"`
	Topics   []topic  `json:"topics"`
}

type reqbodySubscribe struct {
	ClientID string `json:"clientID"`
	Client   client `json:"client"`
}

func (rbSubscribe *reqbodySubscribe) getBrokers() map[string]struct{} {
	brokers := make(map[string]struct{})
	for _, v := range rbSubscribe.Client.Brokers {
		brokers[v] = struct{}{}
	}
	return brokers
}

func (rbSubscribe *reqbodySubscribe) getTopics() map[string]byte {
	topics := make(map[string]byte)
	for _, p := range rbSubscribe.Client.Topics {
		topics[p.Topic] = p.Qos
	}
	return topics
}

type reqbodyUnSubscribe struct {
	ClientID string `json:"clientID"`
}

func loadClients(db *sql.DB) []reqbodySubscribe {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT id, userID, payload FROM public.client WHERE stopped = 'false'")
	tool.CheckThenPanic(err, "load data")
	defer rows.Close()

	clients := make([]reqbodySubscribe, 0)
	for rows.Next() {
		var id, userID, payload string
		err = rows.Scan(&id, &userID, &payload)
		tool.CheckThenPanic(err, "load clients -- row scan")

		var c client
		err = json.Unmarshal([]byte(payload), &c)
		tool.CheckThenPanic(err, "load clients -- json unmarshal")

		clients = append(clients, reqbodySubscribe{
			ClientID: id,
			Client:   c,
		})
	}
	tool.CheckThenPanic(rows.Err(), "load clients")

	return clients
}

// LoadClients load clients from database
func LoadClients(db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqttC.Manager) {
	clients := loadClients(db)

	for _, rb := range clients {
		err := mqttManager.Subscribe(rb.ClientID, rb.Client.Username, rb.Client.Password, rb.getBrokers(), rb.getTopics(), pushFunc(amqpChan, amqpQueue))
		tool.CheckThenPrint(err, "load clients -- client subscribe")
	}
}

// MqttSubscribe mqtt subscription
func MqttSubscribe(db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqttC.Manager) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer func() {
			if err := tool.Error(recover()); err != nil {
				log.Println(err.Error())
				c.JSON(200, gin.H{
					"code":    "no",
					"message": err.Error(),
				})
			}
		}()

		var rb reqbodySubscribe
		err := c.BindJSON(&rb)
		tool.CheckThenPanic(err, "parse request body")

		// TODO get token
		token := ""

		isValid, err := store.ValiClient(db, rb.ClientID, token)
		tool.CheckThenPanic(err, "validate device token")
		if isValid == false {
			panic("invalid device token")
		}

		clientJSON, err := json.Marshal(rb.Client)
		tool.CheckThenPanic(err, "json marshal client")
		err = store.SaveClient(db, rb.ClientID, string(clientJSON))
		tool.CheckThenPanic(err, "store client")

		err = mqttManager.Subscribe(rb.ClientID, rb.Client.Username, rb.Client.Password, rb.getBrokers(), rb.getTopics(), pushFunc(amqpChan, amqpQueue))
		tool.CheckThenPanic(err, "subscribe")

		c.JSON(200, gin.H{
			"code":    "ok",
			"message": "success",
		})
	}
}

// MqttUnSubscribe mqtt unsubscription
func MqttUnSubscribe(db *sql.DB, mqttManager *mqttC.Manager) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer func() {
			if err := tool.Error(recover()); err != nil {
				log.Println(err.Error())
				c.JSON(200, gin.H{
					"code":    "no",
					"message": err.Error(),
				})
			}
		}()

		var rb reqbodyUnSubscribe
		err := c.BindJSON(&rb)
		tool.CheckThenPanic(err, "parse request body")

		err = store.StopClient(db, rb.ClientID)
		tool.CheckThenPanic(err, "stop client in store")

		mqttManager.UnSubscribe(rb.ClientID)

		c.JSON(200, gin.H{
			"code":    "ok",
			"message": "success",
		})
	}
}

// pushFunc push message to message queue
func pushFunc(amqpChan *amqp.Channel, amqpQueue *amqp.Queue) func(clientID string, msg mqtt.Message) {
	return func(clientID string, msg mqtt.Message) {
		sMsg := store.Message{
			ClientID: clientID,
			Topic:    msg.Topic(),
			Payload:  msg.Payload(),
			CreateAt: time.Now(),
		}
		bs, err := json.Marshal(sMsg)
		tool.CheckThenPrint(err, "marshal message")

		err = amqpChan.Publish("", amqpQueue.Name, false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        bs,
		})
		tool.CheckThenPanic(err, "publish message")
	}
}

// Pull and process message
func Pull(down chan<- struct{}, db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue) {
	msgs, err := amqpChan.Consume(amqpQueue.Name, "", true, false, false, false, nil)
	tool.CheckThenPanic(err, "register a consumer")

	log.Printf("waiting for messages")
	for msg := range msgs {
		log.Printf("received a message: %s", msg.Body)
		var sMsg store.Message
		err := json.Unmarshal(msg.Body, &sMsg)
		tool.ErrorThenPanic(err, "unmarshal store message")

		go func() {
			defer func() {
				if err := tool.Error(recover()); err != nil {
					log.Println(err.Error())
				}
			}()

			store.PersistentMessage(db, &sMsg)
		}()
	}

	close(down)
}
