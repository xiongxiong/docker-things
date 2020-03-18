package controller

import (
	"context"
	"database/sql"
	mqttC "dataservice/connector/mqtt"
	"dataservice/store"
	"dataservice/tool"
	"encoding/json"
	"errors"
	"log"
	"net/http"
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

type clientInfo struct {
	ClientID string `json:"clientID"`
	Client   client `json:"client"`
}

func (rbSubscribe *clientInfo) getBrokers() map[string]struct{} {
	brokers := make(map[string]struct{})
	for _, v := range rbSubscribe.Client.Brokers {
		brokers[v] = struct{}{}
	}
	return brokers
}

func (rbSubscribe *clientInfo) getTopics() map[string]byte {
	topics := make(map[string]byte)
	for _, p := range rbSubscribe.Client.Topics {
		topics[p.Topic] = p.Qos
	}
	return topics
}

func loadClients(db *sql.DB) []clientInfo {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := db.QueryContext(ctx, "SELECT id, userID, payload FROM public.client WHERE stopped = 'false'")
	tool.CheckThenPanic(err, "load data")
	defer rows.Close()

	clients := make([]clientInfo, 0)
	for rows.Next() {
		var id, userID, payload string
		err = rows.Scan(&id, &userID, &payload)
		tool.CheckThenPanic(err, "load clients -- row scan")

		var c client
		err = json.Unmarshal([]byte(payload), &c)
		tool.CheckThenPanic(err, "load clients -- json unmarshal")

		clients = append(clients, clientInfo{
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
				c.AbortWithError(http.StatusInternalServerError, err)
			}
		}()

		var rb clientInfo
		err := c.BindJSON(&rb)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}

		// TODO get token
		token := ""

		isValid, err := store.ValiClient(db, rb.ClientID, token)
		tool.CheckThenPanic(err, "validate device token")
		if isValid == false {
			c.AbortWithError(http.StatusUnauthorized, errors.New("invalid device token"))
		}

		clientJSON, err := json.Marshal(rb.Client)
		tool.CheckThenPanic(err, "json marshal client")
		err = store.SaveClient(db, rb.ClientID, string(clientJSON))
		tool.CheckThenPanic(err, "store client")

		err = mqttManager.Subscribe(rb.ClientID, rb.Client.Username, rb.Client.Password, rb.getBrokers(), rb.getTopics(), pushFunc(amqpChan, amqpQueue))
		tool.CheckThenPanic(err, "subscribe")

		c.JSON(http.StatusOK, nil)
	}
}

// MqttUnSubscribe mqtt unsubscription
func MqttUnSubscribe(db *sql.DB, mqttManager *mqttC.Manager) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer func() {
			if err := tool.Error(recover()); err != nil {
				log.Println(err.Error())
				c.AbortWithError(http.StatusInternalServerError, err)
			}
		}()

		var req struct {
			ClientID string `json:"clientID"`
		}
		err := c.BindJSON(&req)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}

		err = store.StopClient(db, req.ClientID)
		tool.CheckThenPanic(err, "stop client in store")

		mqttManager.UnSubscribe(req.ClientID)

		c.JSON(http.StatusOK, nil)
	}
}

// MqttStatus get client status
func MqttStatus(mqttManager *mqttC.Manager) func(c *gin.Context) {
	return func(c *gin.Context) {
		defer func() {
			if err := tool.Error(recover()); err != nil {
				log.Println(err.Error())
				c.AbortWithError(http.StatusInternalServerError, err)
			}
		}()

		var req struct {
			ClientID string `json:"clientID"`
		}
		err := c.BindJSON(&req)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}

		status, err := mqttManager.GetClientStatus(req.ClientID)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}

		res := struct {
			Status string `json:"status"`
		}{status}
		c.JSON(http.StatusOK, res)
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
