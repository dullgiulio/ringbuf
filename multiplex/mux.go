package multiplex

import (
	"fmt"
	"github.com/dullgiulio/ringbuf"
)

const (
	muxMessageCancel = iota
	muxMessageAdd
	muxMessageRemove
)

type MuxMessageType int

type MuxMessage struct {
	msgType MuxMessageType
	ring    *ringbuf.Ringbuf
}

func newMuxMessageCancel() MuxMessage {
	return MuxMessage{
		msgType: muxMessageCancel,
	}
}

func newMuxMessageAdd(ring *ringbuf.Ringbuf) MuxMessage {
	return MuxMessage{
		msgType: muxMessageAdd,
		ring:    ring,
	}
}

func newMuxMessageRemove(ring *ringbuf.Ringbuf) MuxMessage {
	return MuxMessage{
		msgType: muxMessageRemove,
		ring:    ring,
	}
}

type Mux struct {
	messageCh chan MuxMessage
	dataCh    chan interface{}
	rings     []*ringbuf.Ringbuf
	running   bool
}

func NewMux() *Mux {
	return &Mux{
		messageCh: make(chan MuxMessage),
		dataCh:    make(chan interface{}),
		rings:     make([]*ringbuf.Ringbuf, 0),
	}
}

func (m *Mux) String() string {
	return fmt.Sprintf("Mux@%p", m)
}

func (m *Mux) Write(data interface{}) {
	m.dataCh <- data
}

func (m *Mux) Cancel() {
	m.messageCh <- newMuxMessageCancel()
}

func (m *Mux) Add(ring *ringbuf.Ringbuf) {
	m.messageCh <- newMuxMessageAdd(ring)
}

func (m *Mux) Remove(ring *ringbuf.Ringbuf) {
	m.messageCh <- newMuxMessageRemove(ring)
}

func (m *Mux) findRing(rf *ringbuf.Ringbuf) int {
	for r := range m.rings {
		if m.rings[r] == rf {
			return r
		}
	}

	return -1
}

func (m *Mux) handleMessage(errorCh chan<- error, msg MuxMessage) bool {
	switch msg.msgType {
	case muxMessageCancel:
		return false
	case muxMessageAdd:
		if m.findRing(msg.ring) < 0 {
			m.rings = append(m.rings, msg.ring)
		} else {
			errorCh <- fmt.Errorf("%s: Attempt to insert ringbuf %p that has been inserted already.", m, msg.ring)
		}
	case muxMessageRemove:
		if i := m.findRing(msg.ring); i >= 0 {
			m.rings[i], m.rings[len(m.rings)-1], m.rings = m.rings[len(m.rings)-1], nil, m.rings[:len(m.rings)-1]
		} else {
			errorCh <- fmt.Errorf("%s: Attempt to delete a unregistered ring %p.", m, msg.ring)
		}
	}

	return true
}

func (m *Mux) handleData(errorCh chan<- error, data interface{}) {
	for r := range m.rings {
		if m.rings[r] != nil {
			m.rings[r].Write(data)
		}
	}
}

func (m *Mux) Run(errorCh chan<- error) {
	m.running = true

	for {
		select {
		case msg := <-m.messageCh:
			if !m.handleMessage(errorCh, msg) {
				m.running = false
				return
			}
		case data := <-m.dataCh:
			m.handleData(errorCh, data)
		}
	}
}
