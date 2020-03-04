package main

import (
	"context"
	"dataservice/connector/mqtt"
	"dataservice/store"
	"dataservice/tool"
	"log"
	"time"

	gin "github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type reqbodySubscribe struct {
	clientID string
	client   struct {
		username string
		password string
		brokers  []string
		topics   []struct {
			topic string
			qos   byte
		}
	}
}

type reqbodyUnSubscribe struct {
	clientID string
}

func (_global *global) mqttSubscribe(c *gin.Context) {
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

	if len(rb.clientID) != 0 {
		isValid := store.ValidateClientID("", rb.clientID)
		if isValid == false {
			c.JSON(200, gin.H{
				"code":    "no",
				"message": "invalid client id",
			})
		}
	} else {
		rb.clientID = uuid.New().String()
	}

	err = store.SaveClient("")
	tool.CheckThenPanic(err, "store client")

	err = mqtt.Subscribe(string(broker), string(topic), _global.push)
	tool.CheckThenPanic(err, "subscribe")

	c.JSON(200, gin.H{
		"code":    "ok",
		"message": "success",
	})
}

func (_global *global) mqttUnSubscribe(c *gin.Context) {
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

	mqtt.UnSubscribe(rb.clientID)

	c.JSON(200, gin.H{
		"code":    "ok",
		"message": "success",
	})
}

// push message to message queue
func (_global *global) push(topic, message string) {
	q, err := _global.amqpChan.QueueDeclare("hello", false, false, false, false, nil)
	tool.CheckThenPanic(err, "declare a queue")

	err = _global.amqpChan.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	})
	log.Printf(" [x] Sent %s", message)
	tool.CheckThenPanic(err, "publish a message")
}

// pull and process message
func (_global *global) pull() {
	q, err := _global.amqpChan.QueueDeclare("hello", false, false, false, false, nil)
	tool.CheckThenPanic(err, "declare a queue")

	msgs, err := _global.amqpChan.Consume(q.Name, "", true, false, false, false, nil)
	tool.CheckThenPanic(err, "register a consumer")

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("received a message: %s", msg.Body)
			go _global.persistentMessage(string(msg.Body))
		}
	}()

	log.Printf("waiting for messages, to exit press CTRL+C")
	<-forever
}

// persistentMessage persistent message to database
func (_global *global) persistentMessage(message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("the message -- " + message)
	_, err := _global.pgPool.ExecContext(ctx, `insert into t_message (message) values ($1);`, message)
	tool.CheckThenPrint(err, "persistent message")
}
