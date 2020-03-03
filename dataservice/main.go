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

var _Global global

func main() {
	_Global.loadConfig()
	defer _Global.initResource()()
	_Global.loadData()

	go _Global.pull()
	_Global.serve()
}

// read config
func (_global *global) loadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		tool.CheckThenPrint(err, "read config file")
	}

	viper.SetDefault("server.port", "8000")
	_global.serverPort = viper.GetString("server.port")
	log.Printf("config of server port -- %s", _global.serverPort)

	viper.SetDefault("postgres.user", "guest")
	viper.SetDefault("postgres.pass", "guest")
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("postgres.db", "thingspanel")
	_global.pgConnStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", viper.GetString("postgres.user"), viper.GetString("postgres.pass"), viper.GetString("postgres.host"), viper.GetString("postgres.port"), viper.GetString("postgres.db"))
	log.Printf("config of postgres -- %s", _global.pgConnStr)

	viper.SetDefault("amqp.user", "guest")
	viper.SetDefault("amqp.pass", "guest")
	viper.SetDefault("amqp.host", "localhost")
	viper.SetDefault("amqp.port", "5672")
	_global.amqpConnStr = fmt.Sprintf("amqp://%s:%s@%s:%s/", viper.GetString("amqp.user"), viper.GetString("amqp.pass"), viper.GetString("amqp.host"), viper.GetString("amqp.port"))
	log.Printf("config of amqp -- %s", _global.amqpConnStr)
}

func (_global *global) loadData() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// _, err := _global.pgPool.ExecContext(ctx, `insert into messages (msg) values ($1);`, message)
	// tool.CheckThenPrint(err, "persistent message")
}

// init resources
func (_global *global) initResource() (freeFunc func()) {
	log.Println("Prepare resources")

	var err error
	var freeSteps list.List

	for i := 0; i < 10; i++ {
		_global.pgPool, err = sql.Open("postgres", _global.pgConnStr)
		if err != nil {
			log.Println("open postgres failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.CheckThenPanic(err, "open data source")
	freeSteps.PushBack(func() {
		if _global.amqpChan != nil {
			tool.CheckThenPrint(_global.amqpChan.Close(), "close amqp channel")
		}
	})
	_global.pgPool.SetConnMaxLifetime(0)
	_global.pgPool.SetMaxIdleConns(3)
	_global.pgPool.SetMaxOpenConns(3)

	for i := 0; i < 10; i++ {
		_global.amqpConn, err = amqp.Dial(_global.amqpConnStr)
		if err != nil {
			log.Println("open rabbitmq failure, retry after 3 seconds")
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	tool.CheckThenPanic(err, "connect amqp")
	freeSteps.PushBack(func() {
		if _global.amqpConn != nil {
			tool.CheckThenPrint(_global.amqpConn.Close(), "close amqp connection")
		}
	})
	_global.amqpChan, err = _global.amqpConn.Channel()
	tool.CheckThenPanic(err, "open a channel")
	freeSteps.PushBack(func() {
		if _global.pgPool != nil {
			tool.CheckThenPrint(_global.pgPool.Close(), "close data source")
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
func (_global *global) serve() {
	router := gin.Default()

	router.GET("/ping", ping)

	router.POST("/mqtt/unsubscribe", _global.mqttUnSubscribe)
	router.POST("/mqtt/subscribe", _global.mqttSubscribe)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", _global.serverPort),
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

func gracefullyShutdown(srv *http.Server, down chan struct{}) {
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		tool.CheckThenPanic(err, "server shutdown")
	}

	log.Println("server gracefully shutdown")
	close(down)
}

func ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
