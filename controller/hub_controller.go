package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/puertigris/sfu-ws/pkg/hub"
	"log"
	"time"
)

type hubController struct{}

func NewHubController() *hubController {
	return &hubController{}
}

func (_this *hubController) HandlerWs(ctx *gin.Context) {
	roomId := ctx.Param("roomId")
	log.Println("main roomId: ", roomId)
	if _, ok := hub.GetHubManager().HubPool[roomId]; !ok {
		h := hub.NewHub(roomId)
		hub.GetHubManager().Register(h)
		time.Sleep(2 * time.Second)
		log.Println("New hub: ", h.Id)
	}
	hub.GetHubManager().HubPool[roomId].ServeWs(ctx.Writer, ctx.Request)
}
