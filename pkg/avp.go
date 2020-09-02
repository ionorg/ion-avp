package avp

import (
	"context"
	"sync"
	"time"

	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/webrtc/v3"
)

const (
	statCycle = 5 * time.Second
)

var registry *Registry

// AVP represents an avp instance
type AVP struct {
	config  Config
	webrtc  WebRTCTransportConfig
	clients map[string]*SFU
	mu      sync.RWMutex
}

// Init avp with a registry of elements
func Init(r *Registry) {
	registry = r
}

// NewAVP creates a new avp instance
func NewAVP(c Config) *AVP {
	w := WebRTCTransportConfig{
		configuration: webrtc.Configuration{},
		setting:       webrtc.SettingEngine{},
	}

	a := &AVP{
		config:  c,
		clients: make(map[string]*SFU),
		webrtc:  w,
	}

	log.Init(c.Log.Level)

	var icePortStart, icePortEnd uint16

	if len(c.WebRTC.ICEPortRange) == 2 {
		icePortStart = c.WebRTC.ICEPortRange[0]
		icePortEnd = c.WebRTC.ICEPortRange[1]
	}

	if icePortStart != 0 || icePortEnd != 0 {
		if err := a.webrtc.setting.SetEphemeralUDPPortRange(icePortStart, icePortEnd); err != nil {
			panic(err)
		}
	}

	var iceServers []webrtc.ICEServer
	for _, iceServer := range c.WebRTC.ICEServers {
		s := webrtc.ICEServer{
			URLs:       iceServer.URLs,
			Username:   iceServer.Username,
			Credential: iceServer.Credential,
		}
		iceServers = append(iceServers, s)
	}

	a.webrtc.configuration.ICEServers = iceServers

	log.Debugf("WebRTC config:\n%v", a.webrtc)

	go a.stats()

	return a
}

// Process starts a process for a track.
func (a *AVP) Process(ctx context.Context, addr, pid, sid, tid, eid string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		c = NewSFU(addr, a.webrtc)
		c.OnClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	}

	t := c.GetTransport(sid)
	t.Process(pid, tid, eid)
}

// show all avp stats
func (a *AVP) stats() {
	t := time.NewTicker(statCycle)
	for range t.C {
		info := "\n----------------stats-----------------\n"

		a.mu.RLock()
		if len(a.clients) == 0 {
			a.mu.RUnlock()
			continue
		}

		for _, client := range a.clients {
			info += client.stats()
		}
		a.mu.RUnlock()
		log.Infof(info)
	}
}
