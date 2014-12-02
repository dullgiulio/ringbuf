package ringbuf

type Reader struct {
	ring   *Ringbuf
	pos    int64
	cycles int64
	// Channel to write to.
	outputCh chan RingbufData
	starving chan bool
	readCh   chan interface{}
}

func NewReader(r *Ringbuf) *Reader {
	return &Reader{
		ring: r,
		// Do not buffer the reader's outputCh, that will make
		// contention unbearably slow.
		outputCh: make(chan RingbufData),
		starving: make(chan bool),
		readCh:   make(chan interface{}),
	}
}

func (r *Reader) ReadCh() <-chan interface{} {
	go func() {
		for {
			// Request data from the ringbuf. Will reply on outputCh when ready.
			r.ring.dataCh <- newRingbufData(ringbufStatusReader, r)

			// Wait from a response from the ringbuf.
			msg := <-r.outputCh

			switch msg.status {
			case ringbufStatusOK:
				// Write data to our user. Might block.
				r.readCh <- msg.data
			case ringbufStatusEOF:
				// Signal the ringbuf that we are not using it any more.
				r.ring.dataCh <- newRingbufData(ringbufStatusReaderCancel, r)
				return
			case ringbufStatusStarving:
				// The ringbuf has no data. Will signal on this channel that
				// it is ready to serve us if we repeat the request.
				<-r.starving
				continue
			}
		}
	}()

	return r.readCh
}

func (r *Reader) Cancel() {
	r.ring.dataCh <- newRingbufData(ringbufStatusReaderRequestCancel, r)
}

func (r *Reader) cleanup() {
	// Close channels.
	close(r.readCh)
	close(r.outputCh)
	close(r.starving)
}
