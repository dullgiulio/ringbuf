package ringbuf

import (
	"fmt"
	"sync"
	"testing"
)

func TestStrangeRingbuf(t *testing.T) {
	// XXX: Zero-sized ringbuf panics. I can't test that, but believe me it does it.

	ring := NewRingbuf(1)
	reader := NewRingbufReader(ring)

	ring.write("test0")
	ring.write("test1")

	if val, ok := reader.read(); !ok || val != "test1" {
		t.Error(fmt.Sprintf("Expected value test1, got '%s'", val))
	}
}

func helperTestTwoValues(reader *RingbufReader, wg *sync.WaitGroup, t *testing.T) {
	notFirst := false

	for data := range reader.ReadCh() {
		if data == nil {
			break
		}

		if notFirst {
			if data != "test1" {
				t.Error(fmt.Sprintf("Expected value test1, got '%s'", data))
			}
		} else {
			if data != "test0" {
				t.Error(fmt.Sprintf("Expected value test0, got '%s'", data))
			}
			notFirst = true
		}
	}

	if !notFirst {
		t.Error("Could not read both test1 and test0")
	}

	wg.Done()
}

// This look like how I am going to use the Ringbuf in Pippe.
func TestConcurrentWriteReadWithRange(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(2)

	ring := NewRingbuf(3)
	reader1 := NewRingbufReader(ring)
	reader2 := NewRingbufReader(ring)

	// Write from the multiplexed chan
	go func() {
		ring.Write("test0")
		ring.Write("test1")
		ring.Eof()
	}()

	// Each output has its go-routine
	go helperTestTwoValues(reader1, &wg, t)
	go helperTestTwoValues(reader2, &wg, t)

	go ring.Run()

	wg.Wait()
	ring.Cancel()
}

// TODO: Test WriteOrStarve()
