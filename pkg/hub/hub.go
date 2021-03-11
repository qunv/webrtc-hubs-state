package hub

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"github.com/puertigris/sfu-ws/models"
	"log"
	"net/http"
	"sync"
	"time"
)

type Hub struct {
	Id          string
	Clients     []*models.Client
	Register    chan *models.Client
	Unregister  chan *models.Client
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	listLock    sync.RWMutex
	quitChan    chan bool
}

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func NewHub(id string) *Hub {
	return &Hub{
		Id:          id,
		Register:    make(chan *models.Client),
		Unregister:  make(chan *models.Client),
		Clients:     []*models.Client{},
		trackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
		quitChan:    make(chan bool),
	}
}

func (_this *Hub) Run() {
	log.Println("Hub.Run starting with id ", _this.Id)
	for {
		select {
		case client := <-_this.Register:
			log.Println("Hub.Run new client register: ", client.Id)
			_this.Clients = append(_this.Clients, client)
		case client := <-_this.Unregister:
			for i := range _this.Clients {
				if _this.Clients[i].Id == client.Id {
					_this.Clients = remove(_this.Clients, i)
				}
			}
		case <-_this.quitChan:
			log.Println("Hub.Run receive a quit signal!")
			GetHubManager().Quit(_this.Id)
			log.Println("Hub with id", _this.Id, "is closed!")
			return
		}
	}
}

func (_this *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	unsafeConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	c := &models.ThreadSafeWriter{unsafeConn, sync.Mutex{}}

	defer c.Close() //nolint

	// Create new PeerConnection
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		log.Print(err)
		return
	}

	defer peerConnection.Close() //nolint

	// Accept one audio and one video track incoming
	for _, typ := range []webrtc.RTPCodecType{webrtc.RTPCodecTypeVideo, webrtc.RTPCodecTypeAudio} {
		if _, err = peerConnection.AddTransceiverFromKind(typ, webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		}); err != nil {
			log.Print(err)
			return
		}
	}

	// Add our new PeerConnection to global list
	client := models.Client{
		Id: uuid.New().String(),
		Conn: models.PeerConnectionState{
			PeerConnection: peerConnection,
			Websocket:      c,
		},
	}
	_this.listLock.Lock()
	_this.Register <- &client
	_this.listLock.Unlock()

	// Trickle ICE. Emit server candidate to client
	peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
		if i == nil {
			return
		}

		candidateString, err := json.Marshal(i.ToJSON())
		if err != nil {
			log.Println(err)
			return
		}

		if writeErr := c.WriteJSON(&models.WebsocketMessage{
			Event: "candidate",
			Data:  string(candidateString),
		}); writeErr != nil {
			log.Println(writeErr)
		}
	})

	// If PeerConnection is closed remove it from global list
	peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
		switch p {
		case webrtc.PeerConnectionStateFailed:
			if err := peerConnection.Close(); err != nil {
				log.Print(err)
			}
		case webrtc.PeerConnectionStateClosed:
			_this.signalPeerConnections()
		}
		if len(_this.Clients) == 0 {
			_this.quitChan <- true
			return
		}
	})

	peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		// Create a track to fan out our incoming video to all peers
		trackLocal := _this.addTrack(t)
		defer _this.removeTrack(trackLocal)

		buf := make([]byte, 1500)
		for {
			i, _, err := t.Read(buf)
			if err != nil {
				return
			}

			if _, err = trackLocal.Write(buf[:i]); err != nil {
				return
			}
		}
	})

	// Signal for the new PeerConnection
	_this.signalPeerConnections()

	message := &models.WebsocketMessage{}
	for {
		_, raw, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		} else if err := json.Unmarshal(raw, &message); err != nil {
			log.Println(err)
			return
		}

		switch message.Event {
		case "candidate":
			candidate := webrtc.ICECandidateInit{}
			if err := json.Unmarshal([]byte(message.Data), &candidate); err != nil {
				log.Println(err)
				return
			}

			if err := peerConnection.AddICECandidate(candidate); err != nil {
				log.Println(err)
				return
			}
		case "answer":
			answer := webrtc.SessionDescription{}
			if err := json.Unmarshal([]byte(message.Data), &answer); err != nil {
				log.Println(err)
				return
			}

			if err := peerConnection.SetRemoteDescription(answer); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func remove(s []*models.Client, i int) []*models.Client {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

// Add to list of tracks and fire renegotation for all PeerConnections
func (_this *Hub) addTrack(t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	_this.listLock.Lock()
	defer func() {
		_this.listLock.Unlock()
		_this.signalPeerConnections()
	}()

	// Create a new TrackLocal with the same codec as our incoming
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}

	_this.trackLocals[t.ID()] = trackLocal
	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections
func (_this *Hub) removeTrack(t *webrtc.TrackLocalStaticRTP) {
	_this.listLock.Lock()
	defer func() {
		_this.listLock.Unlock()
		_this.signalPeerConnections()
	}()

	delete(_this.trackLocals, t.ID())
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call
func (_this *Hub) dispatchKeyFrame() {
	_this.listLock.Lock()
	defer _this.listLock.Unlock()

	for _, client := range _this.Clients {
		for _, receiver := range client.Conn.PeerConnection.GetReceivers() {
			if receiver.Track() == nil {
				continue
			}

			_ = client.Conn.PeerConnection.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(receiver.Track().SSRC()),
				},
			})
		}
	}
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks
func (_this *Hub) signalPeerConnections() {
	_this.listLock.Lock()
	defer func() {
		_this.listLock.Unlock()
		_this.dispatchKeyFrame()
	}()

	attemptSync := func() (tryAgain bool) {
		for i, client := range _this.Clients {
			if client.Conn.PeerConnection.ConnectionState() == webrtc.PeerConnectionStateClosed {
				_this.Clients = append(_this.Clients[:i], _this.Clients[i+1:]...)
				return true // We modified the slice, start from the beginning
			}

			// map of sender we already are seanding, so we don't double send
			existingSenders := map[string]bool{}

			for _, sender := range client.Conn.PeerConnection.GetSenders() {
				if sender.Track() == nil {
					continue
				}

				existingSenders[sender.Track().ID()] = true

				// If we have a RTPSender that doesn't map to a existing track remove and signal
				if _, ok := _this.trackLocals[sender.Track().ID()]; !ok {
					if err := client.Conn.PeerConnection.RemoveTrack(sender); err != nil {
						return true
					}
				}
			}

			// Don't receive videos we are sending, make sure we don't have loopback
			for _, receiver := range client.Conn.PeerConnection.GetReceivers() {
				if receiver.Track() == nil {
					continue
				}

				existingSenders[receiver.Track().ID()] = true
			}

			// Add all track we aren't sending yet to the PeerConnection
			for trackID := range _this.trackLocals {
				if _, ok := existingSenders[trackID]; !ok {
					if _, err := client.Conn.PeerConnection.AddTrack(_this.trackLocals[trackID]); err != nil {
						return true
					}
				}
			}

			offer, err := client.Conn.PeerConnection.CreateOffer(nil)
			if err != nil {
				return true
			}

			if err = client.Conn.PeerConnection.SetLocalDescription(offer); err != nil {
				return true
			}

			offerString, err := json.Marshal(offer)
			if err != nil {
				return true
			}

			if err = client.Conn.Websocket.WriteJSON(&models.WebsocketMessage{
				Event: "offer",
				Data:  string(offerString),
			}); err != nil {
				return true
			}
		}

		return
	}

	for syncAttempt := 0; ; syncAttempt++ {
		if syncAttempt == 25 {
			// Release the lock and attempt a sync in 3 seconds. We might be blocking a RemoveTrack or AddTrack
			go func() {
				time.Sleep(time.Second * 3)
				_this.signalPeerConnections()
			}()
			return
		}

		if !attemptSync() {
			break
		}
	}
}
