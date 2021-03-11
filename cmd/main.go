package main

import (
	"github.com/puertigris/sfu-ws/controller"
	"log"
)
import "github.com/gin-gonic/gin"

var (
	hubChan chan *controller.Hub
	hubPool map[string]*controller.Hub
)

func StartHub() {
	log.Println("StartHub processing: ", len(hubPool))
	go func() {
		for {
			select {
			case hub := <-hubChan:
				log.Println("Prepare to start hub with id", hub.Id)
				go hub.Run()
			}
		}
	}()
}

func init() {
	hubPool = make(map[string]*controller.Hub)
}

func main() {
	StartHub()

	router := gin.New()
	router.LoadHTMLFiles("index.html")

	router.GET("/room/:roomId", func(c *gin.Context) {
		roomId := c.Param("roomId")
		log.Println("main roomId: ", roomId)
		if _, ok := hubPool[roomId]; !ok {
			hub := controller.NewHub(roomId)
			hubPool[roomId] = hub
			hubPool[roomId].Run()
			log.Println("New hub: ", hub.Id)
		}
		c.HTML(200, "index.html", nil)
	})

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, "HelloWord")
	})

	router.GET("/ws/:roomId", func(c *gin.Context) {
		roomId := c.Param("roomId")
		hubPool[roomId].ServeWs(c.Writer, c.Request)
	})

	router.Run("0.0.0.0:8080")
}
