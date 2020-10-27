package server

import (
	"context"
	"sync"

	avp "github.com/pion/ion-avp/pkg"
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

	return a
}

// Process starts a process for a track.
func (a *AVP) Process(ctx context.Context, addr, pid, sid, tid, eid string, config []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		var err error
		if c, err = NewSFU(addr, a.config); err != nil {
			return err
		}
		c.OnClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	}

	t, err := c.GetTransport(sid)
	if err != nil {
		return err
	}

	return t.Process(pid, tid, eid, config)
}
