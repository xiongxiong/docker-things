package main

import (
	"container/list"
	"context"
	"database/sql"
	"dataservice/connector/mqtt"
	mqttC "dataservice/connector/mqtt"
	"dataservice/controller"
	"dataservice/tool"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	gin "github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

func main() {
	down := make(chan struct{})

	serverPort, pgConnStr, amqpConnStr := loadConfig()
	clean, db, amqpChan, amqpQueue := build(serverPort, pgConnStr, amqpConnStr)
	mqttManager := mqtt.NewManager()

	loadData(db, amqpChan, amqpQueue, mqttManager)

	go controller.Pull(down, db, amqpChan, amqpQueue)

	serve(serverPort, down, clean, db, amqpChan, amqpQueue, mqttManager)
}

func loadConfig() (serverPort, pgConnStr, amqpConnStr string) {
	log.Println("read config")

	var err error

	configFile := flag.String("config", "./config.yml", "path of configuration file")
	flag.Parse()

	viper.SetConfigFile(*configFile)
	if err = viper.ReadInConfig(); err != nil {
		tool.CheckThenPrint(err, "read config file")
	}

	serverPort = viper.GetString("server.port")
	log.Printf("config of server port -- %s", serverPort)

	pgConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", viper.GetString("postgres.user"), viper.GetString("postgres.pass"), viper.GetString("postgres.host"), viper.GetString("postgres.port"), viper.GetString("postgres.db"))
	log.Printf("config of postgres -- %s", pgConnStr)

	amqpConnStr = fmt.Sprintf("amqp://%s:%s@%s:%s/", viper.GetString("amqp.user"), viper.GetString("amqp.pass"), viper.GetString("amqp.host"), viper.GetString("amqp.port"))
	log.Printf("config of amqp -- %s", amqpConnStr)

	return
}

// build resources
func build(serverPort, pgConnStr, amqpConnStr string) (clean func(), db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue) {
	log.Println("build resources")

	var err error
	var cleanSteps list.List

	db, err = sql.Open("postgres", pgConnStr)
	tool.CheckThenPanic(err, "open data source")
	cleanSteps.PushBack(func() {
		if db != nil {
			tool.CheckThenPrint(db.Close(), "close data source")
		}
	})
	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(3)
	db.SetMaxOpenConns(3)

	var amqpConn *amqp.Connection
	amqpConn, err = amqp.Dial(amqpConnStr)
	tool.CheckThenPanic(err, "connect amqp")
	cleanSteps.PushBack(func() {
		if amqpConn != nil {
			tool.CheckThenPrint(amqpConn.Close(), "close amqp connection")
		}
	})

	amqpChan, err = amqpConn.Channel()
	tool.CheckThenPanic(err, "open a channel")
	cleanSteps.PushBack(func() {
		if amqpChan != nil {
			tool.CheckThenPrint(amqpChan.Close(), "close amqp channel")
		}
	})

	var _amqpQueue amqp.Queue
	_amqpQueue, err = amqpChan.QueueDeclare("postgres", true, false, false, false, nil)
	tool.CheckThenPanic(err, "declare a queue")
	amqpQueue = &_amqpQueue

	clean = func() {
		log.Println("Release resources")
		for freeStep := cleanSteps.Back(); freeStep != nil; freeStep = freeStep.Prev() {
			if fc, ok := freeStep.Value.(func()); ok {
				fc()
			}
		}
	}

	return
}

func loadData(db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqttC.Manager) {
	log.Println("load data")

	controller.LoadClients(db, amqpChan, amqpQueue, mqttManager)
}

// serve server
func serve(serverPort string, down chan struct{}, freeFunc func(), db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue, mqttManager *mqtt.Manager) {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin, Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	v1 := r.Group("/api/v1")
	v1.GET("/ping", ping)
	v1.POST("/mqtt/subscribe", controller.MqttSubscribe(db, amqpChan, amqpQueue, mqttManager))
	v1.POST("/mqtt/unsubscribe", controller.MqttUnSubscribe(db, mqttManager))
	v1.POST("/mqtt/status", controller.MqttStatus(mqttManager))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: r,
	}

	go gracefullyShutdown(srv, freeFunc)

	err := srv.ListenAndServe()
	if http.ErrServerClosed != err {
		log.Fatalf("Server not gracefully shutdown, err :%v\n", err)
	}

	<-down
	log.Println("server gracefully shutdown")
}

func gracefullyShutdown(srv *http.Server, freeFunc func()) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		tool.CheckThenPanic(err, "server shutdown")
	}

	freeFunc()
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
