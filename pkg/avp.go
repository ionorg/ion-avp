package avp

import (
	"sync"
	"time"

	"github.com/pion/ion-avp/pkg/log"
)

const (
	statCycle = 3 * time.Second
)

// AVP represents an avp instance
type AVP struct {
	transports map[string]*WebRTCTransport
	mu         sync.RWMutex
}

// NewAVP creates a new avp instance
func NewAVP(c Config) *AVP {
	log.Init(c.Log.Level)

	a := &AVP{
		transports: make(map[string]*WebRTCTransport),
	}

	go a.stats()

	return a
}

// NewWebRTCTransport creates a new webrtctransport for a session
func (a *AVP) NewWebRTCTransport(id string, config Config) *WebRTCTransport {
	t := NewWebRTCTransport(id, config)
	a.mu.Lock()
	a.transports[id] = t
	a.mu.Unlock()
	return t
}

// show all avp stats
func (a *AVP) stats() {
	t := time.NewTicker(statCycle)
	for range t.C {
		info := "\n----------------stats-----------------\n"

		a.mu.RLock()
		if len(a.transports) == 0 {
			a.mu.RUnlock()
			continue
		}

		for _, transport := range a.transports {
			info += transport.stats()
		}
		a.mu.RUnlock()
		log.Infof(info)
		log.Infof(info)
	}
}
