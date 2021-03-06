package main

import (
	"github.com/puertigris/sfu-ws/pkg/hub"
	"github.com/puertigris/sfu-ws/router"
)

func main() {

	hub.NewHubManager().Start()
	r := router.SetupRouters()
	r.LoadHTMLFiles("index.html")
	r.Run("0.0.0.0:8080")
}
