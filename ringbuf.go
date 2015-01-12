package ringbuf

type Ringbuf struct {
	data            []interface{} // type that is stored
	pos             int64         // position for writing
	cycles          int64
	size            int64
	dataCh          chan Data
	readersStarving map[*Reader]bool
	readersCanceled map[*Reader]bool
	readOnly        bool
}

type Write struct {
	data       interface{} // Data to write
	reader     *Reader     // Reader to wait for, if any
	responseCh chan<- bool // Where to confirm the success/failure of the write
}

func NewRingbuf(size int64) *Ringbuf {
	if size <= 0 {
		panic("Tried to allocate a zero-sized Ringbuf")
	}

	return &Ringbuf{
		data:            make([]interface{}, size),
		size:            size,
		dataCh:          make(chan Data),
		readersStarving: make(map[*Reader]bool),
		readersCanceled: make(map[*Reader]bool),
	}
}

// Safe write via channel.
func (r *Ringbuf) Write(data interface{}) {
	r.dataCh <- newData(ringbufStatusWrite, data)
}

func (r *Ringbuf) Cancel() {
	r.dataCh <- newData(ringbufStatusEOF, nil)
}

func (r *Ringbuf) EOF() {
	r.dataCh <- newData(ringbufStatusStarving, nil)
}

func (r *Ringbuf) wakeupStarving() {
	for reader, ok := range r.readersStarving {
		if ok {
			// This reader has been served with data.
			r.readersStarving[reader] = false
			// Tell the reader we have new data, but it
			// will have to be requested again.
			reader.starving <- true
		}
	}
}

func (r *Ringbuf) Run() {
	defer close(r.dataCh)

	for msg := range r.dataCh {
		switch msg.status {
		// Hard quitting of the ringbuf runner.
		case ringbufStatusEOF:
			if len(r.readersStarving) == 0 {
				// When we have exhausted all readers, we can exit.
				// This has the potential to keep this ringbuf open forever
				// if the readers misbehave and don't unsubscribe correctly.
				return
			}

			// Otherwise, just wake up starving readers to send
			// EOF and wait for them to quit gracefully.
			r.wakeupStarving()
		// Writing finished, switch to read-only mode.
		case ringbufStatusStarving:
			r.readOnly = true

			// No more data for starving readers.
			// Wake them up and they'll ask for data, then they'll get EOF.
			r.wakeupStarving()
		// Normal writing.
		case ringbufStatusWrite:
			if !r.readOnly {
				r.write(msg.data)

				// Readers should now try again reading.
				r.wakeupStarving()
			}
			// Reader requesting data.
		case ringbufStatusReader:
			// This is a cast to a pointer, never fails.
			reader := msg.data.(*Reader)

			// This reader has been canceled and must exit.
			if t, ok := r.readersCanceled[reader]; ok && t {
				reader.outputCh <- newData(ringbufStatusEOF, nil)
				continue
			}

			if data, ok := reader.read(); ok && data != nil {
				// Remember this as an active reader, serve it with fresh data.
				r.readersStarving[reader] = false
				reader.outputCh <- newData(ringbufStatusOK, data)
				continue
			}

			if !r.readOnly {
				// This reader is currently starving. Save it so that we can
				// wake it up when we will get new data.
				r.readersStarving[reader] = true
				// Then reply to the reader that we are starving. The reader
				// will then wait until we wake it up via starving channel.
				reader.outputCh <- newData(ringbufStatusStarving, nil)
			} else {
				// We are readOnly (there will be no more writes.) The reader
				// will just get EOF and the reader exits, sending the ReaderCancel
				// message to unsubscribe from this ringbuf.
				r.readersStarving[reader] = false
				reader.outputCh <- newData(ringbufStatusEOF, nil)
			}
		case ringbufStatusReaderRequestCancel:
			reader := msg.data.(*Reader)
			r.readersCanceled[reader] = true

			// If the reader being cancelled is starving, rescue it.
			if r.readersStarving[reader] {
				r.readersStarving[reader] = false
				reader.starving <- true
			}
		// Reader signaling that it has finished reading.
		case ringbufStatusReaderCancel:
			// A reader has finished (either because it is cancelled or got EOF from us)
			// Unregister it from our list of known readers.
			reader := msg.data.(*Reader)
			delete(r.readersStarving, reader)
			delete(r.readersCanceled, reader)

			// Cleanup might take time, do it in the background.
			go reader.cleanup()
		}
	}
}
