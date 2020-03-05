package main

import (
	"container/list"
	"context"
	"database/sql"
	"dataservice/tool"
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

// resources
var db *sql.DB
var amqpConn *amqp.Connection
var amqpChan *amqp.Channel
var amqpQueue amqp.Queue

func main() {
	down := make(chan struct{})

	serverPort, pgConnStr, amqpConnStr := loadConfig()
	drop := _Resources.make(serverPort, pgConnStr, amqpConnStr)
	_Resources.loadData()

	go _Resources.pull(down)
	_Resources.serve(serverPort, down, drop)
}

func loadConfig() (serverPort, pgConnStr, amqpConnStr string) {
	log.Println("read config")

	var err error

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err = viper.ReadInConfig(); err != nil {
		tool.CheckThenPrint(err, "read config file")
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

	return
}

// make resources
func (_resources *resources) make(serverPort, pgConnStr, amqpConnStr string) func() {
	log.Println("make resources")

	var err error
	var dropSteps list.List

	for i := 0; i < 10; i++ {
		_resources.db, err = sql.Open("postgres", pgConnStr)
		if err != nil {
			log.Println("open postgres failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.CheckThenPanic(err, "open data source")
	dropSteps.PushBack(func() {
		if _resources.db != nil {
			tool.CheckThenPrint(_resources.db.Close(), "close data source")
		}
	})
	_resources.db.SetConnMaxLifetime(0)
	_resources.db.SetMaxIdleConns(3)
	_resources.db.SetMaxOpenConns(3)

	for i := 0; i < 10; i++ {
		_resources.amqpConn, err = amqp.Dial(amqpConnStr)
		if err != nil {
			log.Println("open rabbitmq failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.CheckThenPanic(err, "connect amqp")
	dropSteps.PushBack(func() {
		if _resources.amqpConn != nil {
			tool.CheckThenPrint(_resources.amqpConn.Close(), "close amqp connection")
		}
	})

	_resources.amqpChan, err = _resources.amqpConn.Channel()
	tool.CheckThenPanic(err, "open a channel")
	dropSteps.PushBack(func() {
		if _resources.amqpChan != nil {
			tool.CheckThenPrint(_resources.amqpChan.Close(), "close amqp channel")
		}
	})

	_resources.amqpQueue, err = _resources.amqpChan.QueueDeclare("postgres", true, false, false, false, nil)
	tool.CheckThenPanic(err, "declare a queue")

	return func() {
		log.Println("Release resources")

		for freeStep := dropSteps.Back(); freeStep != nil; freeStep = freeStep.Prev() {
			if fc, ok := freeStep.Value.(func()); ok {
				fc()
			}
		}
	}
}

func (_resources *resources) loadData() {
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// _, err := _resources.db.ExecContext(ctx, `insert into messages (msg) values ($1);`, message)
	// tool.CheckThenPrint(err, "persistent message")
}

// serve server
func (_resources *resources) serve(serverPort string, down chan struct{}, freeFunc func()) {
	router := gin.Default()

	router.GET("/ping", ping)

	router.POST("/mqtt/unsubscribe", _resources.mqttUnSubscribe)
	router.POST("/mqtt/subscribe", _resources.mqttSubscribe)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: router,
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
