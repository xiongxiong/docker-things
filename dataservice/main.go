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
// var db *sql.DB
// var amqpConn *amqp.Connection
// var amqpChan *amqp.Channel
// var amqpQueue amqp.Queue

func main() {
	down := make(chan struct{})

	serverPort, pgConnStr, amqpConnStr := loadConfig()
	clean, db, amqpChan, amqpQueue := build(serverPort, pgConnStr, amqpConnStr)
	loadData(db)

	go pull(down, db, amqpChan, amqpQueue)
	serve(serverPort, down, clean, amqpChan, amqpQueue)
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

// build resources
func build(serverPort, pgConnStr, amqpConnStr string) (clean func(), db *sql.DB, amqpChan *amqp.Channel, amqpQueue *amqp.Queue) {
	log.Println("build resources")

	var err error
	var cleanSteps list.List

	// TODO check db is open
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

	// TODO check amqp is open
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

func loadData(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// TODO load data
	_, err := db.ExecContext(ctx, `insert into messages (msg) values ($1);`, "")
	tool.CheckThenPrint(err, "persistent message")
}

// serve server
func serve(serverPort string, down chan struct{}, freeFunc func(), amqpChan *amqp.Channel, amqpQueue *amqp.Queue) {
	router := gin.Default()

	router.GET("/ping", ping)

	router.POST("/mqtt/unsubscribe", mqttUnSubscribe)
	router.POST("/mqtt/subscribe", mqttSubscribe(amqpChan, amqpQueue))

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
