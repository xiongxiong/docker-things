package main

import (
	"container/list"
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

// config
type config struct {
	serverPort, pgConnStr, amqpConnStr string
}

// resource
type resource struct {
	pgPool   *sql.DB
	amqpConn *amqp.Connection
	amqpChan *amqp.Channel
}

// global
type global struct {
	config
	resource
}

var _global global

func main() {
	_global.readConfig()
	defer _global.initResource()()

	go _global.pull()
	_global.serve()
}

// read config
func (glob *global) readConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		tool.CheckThenPrint(err, "read config file")
	}

	viper.SetDefault("server.port", "8000")
	glob.serverPort = viper.GetString("server.port")
	log.Printf("config of server port -- %s", glob.serverPort)

	viper.SetDefault("postgres.user", "guest")
	viper.SetDefault("postgres.pass", "guest")
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("postgres.db", "thingspanel")
	glob.pgConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", viper.GetString("postgres.user"), viper.GetString("postgres.pass"), viper.GetString("postgres.host"), viper.GetString("postgres.port"), viper.GetString("postgres.db"))
	log.Printf("config of postgres -- %s", glob.pgConnStr)

	viper.SetDefault("amqp.user", "guest")
	viper.SetDefault("amqp.pass", "guest")
	viper.SetDefault("amqp.host", "localhost")
	viper.SetDefault("amqp.port", "5672")
	glob.amqpConnStr = fmt.Sprintf("amqp://%s:%s@%s:%s/", viper.GetString("amqp.user"), viper.GetString("amqp.pass"), viper.GetString("amqp.host"), viper.GetString("amqp.port"))
	log.Printf("config of amqp -- %s", glob.amqpConnStr)
}

// init resources
func (glob *global) initResource() (freeFunc func()) {
	log.Println("Prepare resources")

	var err error
	var freeSteps list.List

	for i := 0; i < 10; i++ {
		glob.pgPool, err = sql.Open("postgres", glob.pgConnStr)
		if err != nil {
			log.Println("open postgres failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.CheckThenPanic(err, "open data source")
	freeSteps.PushBack(func() {
		if glob.amqpChan != nil {
			tool.CheckThenPrint(glob.amqpChan.Close(), "close amqp channel")
		}
	})
	glob.pgPool.SetConnMaxLifetime(0)
	glob.pgPool.SetMaxIdleConns(3)
	glob.pgPool.SetMaxOpenConns(3)

	for i := 0; i < 10; i++ {
		glob.amqpConn, err = amqp.Dial(glob.amqpConnStr)
		if err != nil {
			log.Println("open rabbitmq failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.CheckThenPanic(err, "connect amqp")
	freeSteps.PushBack(func() {
		if glob.amqpConn != nil {
			tool.CheckThenPrint(glob.amqpConn.Close(), "close amqp connection")
		}
	})
	glob.amqpChan, err = glob.amqpConn.Channel()
	tool.CheckThenPanic(err, "open a channel")
	freeSteps.PushBack(func() {
		if glob.pgPool != nil {
			tool.CheckThenPrint(glob.pgPool.Close(), "close data source")
		}
	})

	return func() {
		log.Println("Release resources")

		for freeStep := freeSteps.Back(); freeStep != nil; freeStep = freeStep.Prev() {
			if fc, ok := freeStep.Value.(func()); ok {
				fc()
			}
		}
	}
}

// serve server
func (glob *global) serve() {
	router := gin.Default()
	router.GET("/ping", ping)
	router.GET("/connect/mqtt/:broker/:topic", glob.connectMQTT)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", glob.serverPort),
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

func (glob *global) connectMQTT(c *gin.Context) {
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
	tool.CheckThenPanic(err, "connect mqtt broker")
	topic, err := base64.StdEncoding.DecodeString(c.Param("topic"))
	tool.CheckThenPanic(err, "connect mqtt topic")
	err = mqtt.SubBrokerTopic(string(broker), string(topic), glob.push)
	tool.CheckThenPanic(err, "subscribe")

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
func (glob *global) push(topic, message string) {
	q, err := glob.amqpChan.QueueDeclare("hello", false, false, false, false, nil)
	tool.CheckThenPanic(err, "declare a queue")

	err = glob.amqpChan.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message),
	})
	log.Printf(" [x] Sent %s", message)
	tool.CheckThenPanic(err, "publish a message")
}

// pull and process message
func (glob *global) pull() {
	q, err := glob.amqpChan.QueueDeclare("hello", false, false, false, false, nil)
	tool.CheckThenPanic(err, "declare a queue")

	msgs, err := glob.amqpChan.Consume(q.Name, "", true, false, false, false, nil)
	tool.CheckThenPanic(err, "register a consumer")

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
			go glob.persistentMessage(string(msg.Body))
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

// persistentMessage persistent message to database
func (glob *global) persistentMessage(message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	println("the message -- " + message)
	_, err := glob.pgPool.ExecContext(ctx, `insert into message (msg) values ($1);`, message)
	tool.CheckThenPrint(err, "persistent message")
}
