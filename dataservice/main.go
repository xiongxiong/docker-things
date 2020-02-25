package main

import (
	"context"
	"database/sql"
	"dataservice/connector/mqtt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gin "github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

const connStr string = "postgres://guest:guest@localhost/thingspanel?sslmode=disable"

var err error
var pool *sql.DB
var conn *amqp.Connection
var ch *amqp.Channel

func main() {
	prepare()
	defer release()

	go mqtt.SubscribeAll(Push)
	go Pull()
	Serve()
}

// prepare resources
func prepare() {
	log.Println("Prepare resources")

	pool, err = sql.Open("postgres", connStr)
	failOnError(err, "unable to use data source name")
	pool.SetConnMaxLifetime(0)
	pool.SetMaxIdleConns(3)
	pool.SetMaxOpenConns(3)

	conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err = conn.Channel()
	failOnError(err, "Failed to open a channel")
}

// release resources
func release() {
	log.Println("Release resources")

	if pool != nil {
		pool.Close()
	}
	if conn != nil {
		conn.Close()
	}
	if ch != nil {
		ch.Close()
	}
}

// Serve server
func Serve() {
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	router.GET("/connect/mqtt/:broker/:topic", func(c *gin.Context) {
		// broker := c.Param("broker")
		// topic := c.Param("topic")
		go mqtt.SubscribeAll(Push)
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	srv := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}
	down := make(chan struct{})

	go gracefullyShutdown(srv, down)

	err = srv.ListenAndServe()
	if http.ErrServerClosed != err {
		log.Fatalf("Server not gracefully shutdown, err :%v\n", err)
	}

	<-down
}

func gracefullyShutdown(srv *http.Server, down chan struct{}) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown: ", err)
	}

	log.Println("Server gracefully shutdown")
	close(down)
}

// Push push message to message queue
func Push(topic, message string) {
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
	q, err := ch.QueueDeclare("hello", false, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
			go PersistentMessage(string(msg.Body))
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// PersistentMessage persistent message to database
func PersistentMessage(message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	println("the message -- " + message)
	_, err := pool.ExecContext(ctx, `insert into message (msg) values ($1);`, message)
	logOnError(err, "unable to persistent message")
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func logOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}
