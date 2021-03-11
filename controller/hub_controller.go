package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/puertigris/sfu-ws/pkg/hub"
	"log"
)

type hubController struct{}

func NewHubController() *hubController {
	return &hubController{}
}

func (_this *hubController) Join(ctx *gin.Context) {
	roomId := ctx.Param("roomId")
	log.Println("main roomId: ", roomId)
	if _, ok := hub.GetHubManager().HubPool[roomId]; !ok {
		h := hub.NewHub(roomId)
		hub.GetHubManager().Register(h)
		log.Println("New hub: ", h.Id)
	}
	ctx.HTML(200, "index.html", nil)
}

func (_this *hubController) HandlerWs(ctx *gin.Context) {
	roomId := ctx.Param("roomId")
	hub.GetHubManager().HubPool[roomId].ServeWs(ctx.Writer, ctx.Request)
}
