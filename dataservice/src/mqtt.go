package main

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
	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type client struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Brokers  []string `json:"brokers"`
	Topics   []struct {
		Topic string `json:"topic"`
		Qos   byte   `json:"qos"`
	} `json:"topics"`
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

func loadClients(db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqttC.Manager) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT id, userID, payload FROM public.client WHERE stopped = 'false'")
	tool.CheckThenPanic(err, "load data")
	defer rows.Close()

	clients := make([]reqbodySubscribe, 0)
	for rows.Next() {
		var id, userID, payload string
		err = rows.Scan(id, userID, payload)
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

	for _, rb := range clients {
		err = mqttManager.Subscribe(rb.ClientID, rb.Client.Username, rb.Client.Password, rb.getBrokers(), rb.getTopics(), pushFunc(amqpChan, amqpQueue))
		tool.CheckThenPanic(err, "load clients -- client subscribe")
	}
}

func mqttSubscribe(db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqttC.Manager) func(c *gin.Context) {
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
		err := c.BindJSON(rb)
		tool.CheckThenPanic(err, "parse request body")

		if len(rb.ClientID) != 0 {
			// TODO process userID
			isValid, err := store.ValidateClientID(db, "--", rb.ClientID)
			tool.CheckThenPanic(err, "validate client id")
			if isValid == false {
				c.JSON(200, gin.H{
					"code":    "no",
					"message": "invalid client id",
				})
			}
		} else {
			rb.ClientID = uuid.New().String()
		}

		clientJSON, err := json.Marshal(rb.Client)
		tool.CheckThenPanic(err, "json marshal client")
		err = store.SaveClient(db, "--", rb.ClientID, string(clientJSON))
		tool.CheckThenPanic(err, "store client")

		err = mqttManager.Subscribe(rb.ClientID, rb.Client.Username, rb.Client.Password, rb.getBrokers(), rb.getTopics(), pushFunc(amqpChan, amqpQueue))
		tool.CheckThenPanic(err, "subscribe")

		c.JSON(200, gin.H{
			"code":    "ok",
			"message": "success",
		})
	}
}

func mqttUnSubscribe(db *sql.DB, mqttManager *mqttC.Manager) func(c *gin.Context) {
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
		err := c.BindJSON(rb)
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
			ClientID:  clientID,
			Topic:     msg.Topic(),
			Payload:   msg.Payload(),
			CreatedAt: time.Now(),
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

// pull and process message
func pull(down chan<- struct{}, db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue) {
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
