package ringbuf

import (
	"fmt"
	"testing"
)

func TestBytesWriterInterface(t *testing.T) {
	readReady := make(chan bool)

	writer := NewRingbufBytes(1024)

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

		reader := NewRingbufReaderBytes(writer)
		i := 0

		for {
			data := make([]byte, 12)

			// Blocks here until new data is read.
			if n, err := reader.Read(data); err == nil {
				if n == 0 {
					break
				}

				// XXX: What ending do we have to remove here?!
				data = data[0:n]
				exp := fmt.Sprintf("Some data %d", i)

				if string(data) != exp {
					t.Error(fmt.Sprintf("Unexpected read from ringbuf.Reader(): expected '%s', got '%s'", exp, data))
				}
			} else {
				t.Error(fmt.Sprintf("Invalid read in ringbuf.Reader(): error: %s", err))
			}

			i++
		}

		writer.Close()
	}()

	writer.Ringbuf().Run()
	close(readReady)
}
