package controller

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type Client struct {
	Id   string
	Conn PeerConnectionState
}

type websocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)
