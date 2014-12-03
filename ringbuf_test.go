package ringbuf

import (
	"fmt"
	"sync"
	"testing"
)

func TestZeroRingbuf(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic()")
		}
	}()

	NewRingbuf(0)
}

func TestStrangeRingbuf(t *testing.T) {
	ring := NewRingbuf(1)
	reader := NewReader(ring)

	ring.write("test0")
	ring.write("test1")

	if val, ok := reader.read(); ok || val != "test1" {
		t.Error(fmt.Sprintf("Expected value test1, got '%s'", val))
	}
}

func helperTestTwoValues(reader *Reader, wg *sync.WaitGroup, t *testing.T) {
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

func TestConcurrentWriteReadWithRange(t *testing.T) {
	var wg sync.WaitGroup

	wg.Add(2)

	ring := NewRingbuf(3)
	reader1 := NewReader(ring)
	reader2 := NewReader(ring)

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

func TestReadWithStarve(t *testing.T) {
	ring := NewRingbuf(3)
	reader := NewReader(ring)
	readCh := reader.ReadCh()

	go ring.Run()

	proceed := make(chan bool, 1)

	go func() {
		ring.Write("test0")
		s := <-readCh
		if s != "test0" {
			t.Error("Expected string not found")
		}

		ring.Write("test1")
		s = <-readCh
		if s != "test1" {
			t.Error("Expected string not found")
		}

		proceed <- true
		s = <-readCh
	}()

	<-proceed
	reader.Cancel()

	ring.Eof()
	ring.Cancel()
}
