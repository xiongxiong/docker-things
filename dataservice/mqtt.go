package main

import (
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

type reqbodySubscribe struct {
	ClientID string   `json:"clientID"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Brokers  []string `json:"brokers"`
	Topics   []struct {
		Topic string `json:"topic"`
		Qos   byte   `json:"qos"`
	} `json:"topics"`
}

func (rbSubscribe *reqbodySubscribe) getBrokers() map[string]struct{} {
	brokers := make(map[string]struct{})
	for _, v := range rbSubscribe.Brokers {
		brokers[v] = struct{}{}
	}
	return brokers
}

func (rbSubscribe *reqbodySubscribe) getTopics() map[string]byte {
	topics := make(map[string]byte)
	for _, p := range rbSubscribe.Topics {
		topics[p.Topic] = p.Qos
	}
	return topics
}

type reqbodyUnSubscribe struct {
	ClientID string `json:"clientID"`
}

func mqttSubscribe(amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqttC.Manager) func(c *gin.Context) {
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
			isValid := store.ValidateClientID("", rb.ClientID)
			if isValid == false {
				c.JSON(200, gin.H{
					"code":    "no",
					"message": "invalid client id",
				})
			}
		} else {
			rb.ClientID = uuid.New().String()
		}

		err = store.SaveClient("")
		tool.CheckThenPanic(err, "store client")

		err = mqttManager.Subscribe(rb.ClientID, rb.Username, rb.Password, rb.getBrokers(), rb.getTopics(), pushFunc(amqpChan, amqpQueue))
		tool.CheckThenPanic(err, "subscribe")

		c.JSON(200, gin.H{
			"code":    "ok",
			"message": "success",
		})
	}
}

func mqttUnSubscribe(mqttManager *mqttC.Manager) func(c *gin.Context) {
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

		// TODO save postgres

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
