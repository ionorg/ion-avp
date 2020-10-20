package server

import (
	"context"
	"sync"
	"time"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

const (
	statCycle = 5 * time.Second
)

// AVP represents an avp instance
type AVP struct {
	config  avp.Config
	clients map[string]*SFU
	mu      sync.RWMutex
}

// NewAVP creates a new avp instance
func NewAVP(c avp.Config, elems map[string]avp.ElementFun) *AVP {
	a := &AVP{
		config:  c,
		clients: make(map[string]*SFU),
	}

	avp.Init(elems)

	go a.stats()

	return a
}

// Process starts a process for a track.
func (a *AVP) Process(ctx context.Context, addr, pid, sid, tid, eid string, config []byte) {
	a.mu.Lock()
	defer a.mu.Unlock()

	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		c = NewSFU(addr, a.config)
		c.OnClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	}

	t := c.GetTransport(sid)
	t.Process(pid, tid, eid, config)
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
