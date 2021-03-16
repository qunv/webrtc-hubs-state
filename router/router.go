package router

import (
	"github.com/gin-gonic/gin"
	"github.com/puertigris/sfu-ws/controller"
)

func SetupRouters() *gin.Engine {
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	hubController := controller.NewHubController()
	r.GET("/ws/:roomId", hubController.HandlerWs)
	return r
}
