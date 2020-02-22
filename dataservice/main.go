package main

import (
	"dataservice/mqtt"
	"dataservice/rabbit"

	gin "github.com/gin-gonic/gin"
)

func main() {
	go rabbit.Pull()
	go mqtt.SubscribeAll()
	Serv()
}

// Serv server
func Serv() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/connect/mqtt/:broker/:topic", func(c *gin.Context) {
		// broker := c.Param("broker")
		// topic := c.Param("topic")
		// go mqtt.SubscribeAll()
		c.JSON(200, gin.H{
			"message": "ok",
		})
	})
	r.Run()
}
