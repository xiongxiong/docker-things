package main

import (
	"context"
	"database/sql"
	"dataservice/connector/mqtt"
	"dataservice/tool"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gin "github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

var serverPort string
var pgConnStr string
var pgPool *sql.DB
var amqpConnStr string
var amqpConn *amqp.Connection
var amqpChan *amqp.Channel

func main() {
	config()
	prepare()
	defer release()

	go pull()
	serve()
}

// read config
func config() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		tool.PrintError(err, "fatal error of config file, use default setting")
	}

	viper.SetDefault("server.port", "8000")
	serverPort = viper.GetString("server.port")
	log.Printf("config of server port -- %s", serverPort)

	viper.SetDefault("postgres.user", "guest")
	viper.SetDefault("postgres.pass", "guest")
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("postgres.db", "thingspanel")
	pgConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", viper.GetString("postgres.user"), viper.GetString("postgres.pass"), viper.GetString("postgres.host"), viper.GetString("postgres.port"), viper.GetString("postgres.db"))
	log.Printf("config of postgres -- %s", pgConnStr)

	viper.SetDefault("amqp.user", "guest")
	viper.SetDefault("amqp.pass", "guest")
	viper.SetDefault("amqp.host", "localhost")
	viper.SetDefault("amqp.port", "5672")
	amqpConnStr = fmt.Sprintf("amqp://%s:%s@%s:%s/", viper.GetString("amqp.user"), viper.GetString("amqp.pass"), viper.GetString("amqp.host"), viper.GetString("amqp.port"))
	log.Printf("config of amqp -- %s", amqpConnStr)
}

// prepare resources
func prepare() {
	log.Println("Prepare resources")

	var err error
	for i := 0; i < 10; i++ {
		pgPool, err = sql.Open("postgres", pgConnStr)
		if err != nil {
			log.Println("open postgres failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.PanicError(err, "unable to use data source name")
	pgPool.SetConnMaxLifetime(0)
	pgPool.SetMaxIdleConns(3)
	pgPool.SetMaxOpenConns(3)

	for i := 0; i < 10; i++ {
		amqpConn, err = amqp.Dial(amqpConnStr)
		if err != nil {
			log.Println("open rabbitmq failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.PanicError(err, "unable to connect to rabbitmq")
	amqpChan, err = amqpConn.Channel()
	tool.PanicError(err, "unable to open a channel")
}

// release resources
func release() {
	log.Println("Release resources")

	if pgPool != nil {
		pgPool.Close()
	}
	if amqpConn != nil {
		amqpConn.Close()
	}
	if amqpChan != nil {
		amqpChan.Close()
	}
}

// serve server
func serve() {
	router := gin.Default()
	router.GET("/ping", ping)
	router.GET("/connect/mqtt/:broker/:topic", connectMQTT)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: router,
	}
	down := make(chan struct{})

	go gracefullyShutdown(srv, down)

	err := srv.ListenAndServe()
	if http.ErrServerClosed != err {
		log.Fatalf("Server not gracefully shutdown, err :%v\n", err)
	}

	<-down
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func connectMQTT(c *gin.Context) {
	defer func() {
		if err := tool.Error(recover()); err != nil {
			log.Println(err.Error())
			c.JSON(200, gin.H{
				"success": false,
				"message": err.Error(),
			})
		}
	}()

	broker, err := base64.StdEncoding.DecodeString(c.Param("broker"))
	tool.PanicError(err, "connect mqtt invalid broker")
	topic, err := base64.StdEncoding.DecodeString(c.Param("topic"))
	tool.PanicError(err, "connect mqtt invalid topic")
	err = mqtt.SubBrokerTopic(string(broker), string(topic), push)
	tool.PanicError(err, "subscribe error")

	c.JSON(200, gin.H{
		"success": true,
		"message": "success",
	})
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

// push message to message queue
func push(topic, message string) {
	q, err := amqpChan.QueueDeclare("hello", false, false, false, false, nil)
	tool.PanicError(err, "Failed to declare a queue")

	err = amqpChan.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	})
	log.Printf(" [x] Sent %s", message)
	tool.PanicError(err, "Failed to publish a message")
}

// pull and process message
func pull() {
	q, err := amqpChan.QueueDeclare("hello", false, false, false, false, nil)
	tool.PanicError(err, "Failed to declare a queue")

	msgs, err := amqpChan.Consume(q.Name, "", true, false, false, false, nil)
	tool.PanicError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
			go persistentMessage(string(msg.Body))
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// persistentMessage persistent message to database
func persistentMessage(message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	println("the message -- " + message)
	_, err := pgPool.ExecContext(ctx, `insert into message (msg) values ($1);`, message)
	tool.PrintError(err, "unable to persistent message")
}
