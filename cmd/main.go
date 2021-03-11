package main

import (
	"github.com/gin-gonic/gin"
	"github.com/puertigris/sfu-ws/pkg/hub"
	"log"
)

func main() {

	hub.NewHubManager().Start()

	router := gin.New()
	router.LoadHTMLFiles("index.html")

	router.GET("/room/:roomId", func(c *gin.Context) {
		roomId := c.Param("roomId")
		log.Println("main roomId: ", roomId)
		if _, ok := hub.GetHubManager().HubPool[roomId]; !ok {
			h := hub.NewHub(roomId)
			hub.GetHubManager().Register(h)
			log.Println("New hub: ", h.Id)
		}
		c.HTML(200, "index.html", nil)
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, "HelloWord")
	})

	router.GET("/ws/:roomId", func(c *gin.Context) {
		roomId := c.Param("roomId")
		hub.GetHubManager().HubPool[roomId].ServeWs(c.Writer, c.Request)
	})

	router.Run("0.0.0.0:8080")
}
