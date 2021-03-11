package hub

import (
	"log"
	"sync"
)

type HubManager struct {
	hubChan     chan *Hub
	quitHubChan chan string
	HubPool     map[string]*Hub //hubId <-> Hub
}

var hubManager *HubManager = nil
var hubManagerOnce = sync.Once{}

func NewHubManager() *HubManager {
	hubManagerOnce.Do(func() {
		hubManager = &HubManager{
			hubChan:     make(chan *Hub),
			quitHubChan: make(chan string),
			HubPool:     make(map[string]*Hub),
		}
	})
	return hubManager
}

func GetHubManager() *HubManager {
	return hubManager
}

func (_this *HubManager) Start() {
	log.Println("HubManager start!")
	go func() {
		for {
			select {
			case hub := <-_this.hubChan:
				log.Println("HubManager.Start prepare for starting hub with id: ", hub.Id)
				_this.HubPool[hub.Id] = hub
				go hub.Run()
			case hubId := <-_this.quitHubChan:
				log.Println("HubManager.Start receive a quit signal for hub id: ", hubId)
				if _, ok := _this.HubPool[hubId]; !ok {
					log.Println("HubManager.Start hub with id ", hubId, " is not existed!")
				} else {
					delete(_this.HubPool, hubId)
					log.Println("HubManager.Start removed hub id: ", hubId, "from hub-pool")
				}
			}

		}
	}()
}

func (_this *HubManager) Register(hub *Hub) {
	_this.hubChan <- hub
}

func (_this *HubManager) Quit(hubId string) {
	_this.quitHubChan <- hubId
}
