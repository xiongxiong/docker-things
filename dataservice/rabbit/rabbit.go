package rabbit

import (
	"log"

	"github.com/streadway/amqp"
)

var conn *amqp.Connection
var ch *amqp.Channel

// Push push message to message queue
func Push(topic, message string) {
	if conn == nil {
		conn, _ = amqp.Dial("amqp://guest:guest@localhost:5672/")
		// failOnError(err, "Failed to connect to RabbitMQ")
		// defer conn.Close()
	}

	if ch == nil {
		ch, _ = conn.Channel()
		// failOnError(err, "Failed to open a channel")
		// defer ch.Close()
	}

	q, err := ch.QueueDeclare("hello", false, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	err = ch.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	})
	log.Printf(" [x] Sent %s", message)
	failOnError(err, "Failed to publish a message")
}

// Pull pull and process message
func Pull() {
	if conn == nil {
		conn, _ = amqp.Dial("amqp://guest:guest@localhost:5672/")
		// failOnError(err, "Failed to connect to RabbitMQ")
		// defer conn.Close()
	}

	if ch == nil {
		ch, _ = conn.Channel()
		// failOnError(err, "Failed to open a channel")
		// defer ch.Close()
	}

	q, err := ch.QueueDeclare("hello", false, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
