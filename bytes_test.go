package ringbuf

import (
	"fmt"
	"testing"
)

func _testBytesWriter(t *testing.T, writer *RingbufBytes) {
	readReady := make(chan bool)

	go func() {
		for i := 0; i < 20; i++ {
			writer.Write([]byte(fmt.Sprintf("Some data %d", i)))
		}

		writer.Eof()
		readReady <- true
	}()

	go func() {
		// Wait for all writes to be completed.
		<-readReady

		reader := NewReaderBytes(writer)
		data := make([]byte, 100)
		smallbuf := make([]byte, 2)

		if n, err := reader.Read(smallbuf); n != 2 && err != nil {
			t.Error("Expected error because buffer is too small")
		}

		i := 1

		for {
			// Blocks here until new data is read.
			if n, err := reader.Read(data); err == nil {
				if n == 0 {
					break
				}

				str := string(data[0:n])
				exp := fmt.Sprintf("Some data %d", i)

				if str != exp {
					t.Error(fmt.Sprintf("Unexpected read from ringbuf.Reader(): expected '%s', got '%s'", exp, data))
				}
			} else {
				t.Error(fmt.Sprintf("Invalid read in ringbuf.Reader(): error: %s", err))
			}

			i++
		}

		writer.Close()

		if n, e := reader.Read(data); n != 0 && e == nil {
			t.Error("Did not expect a valid read after EOF")
		}
	}()

	writer.Ringbuf().Run()
	close(readReady)
}

func TestNewRinbufWriter(t *testing.T) {
	writer := NewRingbufBytes(1024)
	_testBytesWriter(t, writer)
}

func TestBytesWriter(t *testing.T) {
	ring := NewRingbuf(1024)
	writer := NewBytes(ring)
	_testBytesWriter(t, writer)
}
