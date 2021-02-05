package server

import (
	"context"
	"fmt"
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

	t, err := a.getTransport(addr, sid, config)
	if err != nil {
		return err
	}

	return t.Process(pid, tid, eid, config)
}

func (a *AVP) Run(addr, sid, tid string, element avp.Element) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	t, err := a.getTransport(addr, sid, nil)
	if err != nil {
		return err
	}

	return t.Run(tid, element)
}

// Stop stops processing a track. Call when Process or Run should end.
func (a *AVP) Stop(addr, sid, tid string) error {
	t, err := a.getTransportLocked(addr, sid, nil)
	if err != nil {
		return err
	}

	t.Stop(tid)
	return nil
}

func (a *AVP) getTransportLocked(addr, sid string, config []byte) (*avp.WebRTCTransport, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.getTransport(addr, sid, nil)
}

func (a *AVP) getTransport(addr, sid string, config []byte) (*avp.WebRTCTransport, error) {
	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		var err error
		if c, err = NewSFU(addr, a.config); err != nil {
			return nil, err
		}
		c.OnClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	}

	return c.GetTransport(sid)
}

// CloseSession stops all processing for a session. Call it when the session ends.
func (a *AVP) CloseSession(addr, sid string) error {
	c := a.clients[addr]
	if c == nil {
		return fmt.Errorf("missing grpc client for %s", addr)
	}
	t, err := c.GetTransport(sid)
	if err != nil {
		return fmt.Errorf("err GetTransport for session %s: %w", sid, err)
	}
	if t == nil {
		return fmt.Errorf("missing transport for session %s", sid)
	}
	return t.Close()
}
