package multiplex

import (
	"bitbucket.org/dullgiulio/ringbuf"
	"fmt"
)

const (
	demuxMessageCancel = iota
	demuxMessageAdd
	demuxMessageRemove
)

type DemuxMessageType int

type DemuxMessage struct {
	msgType DemuxMessageType
	reader  *DemuxReader
}

func newDemuxMessageCancel() DemuxMessage {
	return DemuxMessage{
		msgType: demuxMessageCancel,
	}
}

func newDemuxMessageAdd(reader *DemuxReader) DemuxMessage {
	return DemuxMessage{
		msgType: demuxMessageAdd,
		reader:  reader,
	}
}

func newDemuxMessageRemove(reader *DemuxReader) DemuxMessage {
	return DemuxMessage{
		msgType: demuxMessageRemove,
		reader:  reader,
	}
}

type DemuxReader struct {
	reader   *ringbuf.RingbufReader
	cancelCh chan bool
	onCancel func()
}

func NewDemuxReader(reader *ringbuf.RingbufReader) *DemuxReader {
	return &DemuxReader{
		reader:   reader,
		cancelCh: make(chan bool),
	}
}

func (dr *DemuxReader) Cancel() {
	dr.cancelCh <- true
}

func (dr *DemuxReader) SetOnCancel(f func()) {
	dr.onCancel = f
}

func (dr *DemuxReader) Run(ring *ringbuf.Ringbuf) {
	readCh := dr.reader.ReadCh()
	readOnly := false

	defer func() {
		if dr.onCancel != nil {
			dr.onCancel()
		}
	}()

	for {
		select {
		case <-dr.cancelCh:
			dr.reader.Cancel()
			readOnly = true
		case data := <-readCh:
			if data == nil {
				return
			}

			if !readOnly {
				ring.Write(data)
			}
		}
	}
}

type Demux struct {
	messageCh chan DemuxMessage
	dataCh    chan interface{}
	readers   []*DemuxReader
	ring      *ringbuf.Ringbuf
}

func NewDemux() *Demux {
	return &Demux{
		messageCh: make(chan DemuxMessage),
		dataCh:    make(chan interface{}),
		ring:      ringbuf.NewRingbuf(1024),
		readers:   make([]*DemuxReader, 0),
	}
}

func (d *Demux) String() string {
	return fmt.Sprintf("Demux@%p", d)
}

func (d *Demux) Cancel() {
	d.messageCh <- newDemuxMessageCancel()
}

func (d *Demux) Add(reader *DemuxReader) {
	d.messageCh <- newDemuxMessageAdd(reader)
}

func (d *Demux) findReader(rr *DemuxReader) int {
	for r := range d.readers {
		if d.readers[r] == rr {
			return r
		}
	}

	return -1
}

func (d *Demux) handleMessage(errorCh chan<- error, msg DemuxMessage) bool {
	switch msg.msgType {
	case demuxMessageCancel:
		return false
	case demuxMessageAdd:
		if d.findReader(msg.reader) < 0 {
			go msg.reader.Run(d.ring)

			d.readers = append(d.readers, msg.reader)
		} else {
			errorCh <- fmt.Errorf("%s: Attempt to insert reader %p that been inserted already.", d, msg.reader)
		}
	case demuxMessageRemove:
		if i := d.findReader(msg.reader); i >= 0 {
			d.readers[i], d.readers[len(d.readers)-1], d.readers = d.readers[len(d.readers)-1], nil, d.readers[:len(d.readers)-1]
		} else {
			errorCh <- fmt.Errorf("%s: Attempt to delete a unregistered reader %p.", d, msg.reader)
		}
	}

	return true
}

func (d *Demux) Reader() *ringbuf.RingbufReader {
	return ringbuf.NewRingbufReader(d.ring)
}

func (d *Demux) Run(errorCh chan<- error) {
	go d.ring.Run()

	for msg := range d.messageCh {
		if !d.handleMessage(errorCh, msg) {
			// Cancel all readers.
			for r := range d.readers {
				d.readers[r].Cancel()
			}

			d.ring.Cancel()
			break
		}
	}
}
