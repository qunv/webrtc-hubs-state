package models

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"sync"
)

type Client struct {
	Id   string
	Conn PeerConnectionState
}

type WebsocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type PeerConnectionState struct {
	PeerConnection *webrtc.PeerConnection
	Websocket      *ThreadSafeWriter
}

type ThreadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *ThreadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}
