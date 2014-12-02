package ringbuf

import "fmt"

// Implement reader and writer interface for []byte

type RingbufBytes struct {
	r *Ringbuf
}

func NewRingbufBytes(size int64) *RingbufBytes {
	return &RingbufBytes{
		r: NewRingbuf(size),
	}
}

func NewBytes(r *Ringbuf) *RingbufBytes {
	return &RingbufBytes{r: r}
}

func (rb *RingbufBytes) Write(b []byte) (int, error) {
	data := make([]byte, len(b))
	copy(data, b)

	rb.r.Write(data)
	return len(data), nil
}

func (rb *RingbufBytes) Close() {
	rb.r.Cancel()
}

func (rb *RingbufBytes) Eof() {
	rb.r.Eof()
}

func (rb *RingbufBytes) Ringbuf() *Ringbuf {
	return rb.r
}

type ReaderBytes struct {
	rb    *Reader
	ch    <-chan interface{}
	isEOF bool
}

func NewReaderBytes(r *RingbufBytes) *ReaderBytes {
	reader := NewReader(r.r)
	return &ReaderBytes{
		rb: reader,
		ch: reader.ReadCh(),
	}
}

func (r *ReaderBytes) Read(p []byte) (bread int, err error) {
	if r.isEOF {
		return
	}

	// This will block until there is unread data to read.
	data := <-r.ch

	if bytes, ok := data.([]byte); ok {
		bread = len(bytes)
		size := len(p)

		if bread > size {
			err = fmt.Errorf("Given size %d is too small. Read %d bytes.", size, bread)
			bread = size
		}

		copy(p, bytes)
	} else {
		r.isEOF = true
	}

	return
}
