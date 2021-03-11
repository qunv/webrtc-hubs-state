package controller

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"sync"
)

type WebsocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type PeerConnectionState struct {
	peerConnection *webrtc.PeerConnection
	websocket      *threadSafeWriter
}

type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}
