package multiplex

import (
	"fmt"
	"github.com/dullgiulio/ringbuf"
	"testing"
)

func TestMuxCancel(t *testing.T) {
	mux := NewMux()

	if mux.running == true {
		t.Error("Newly created mux is already marked as running.")
	}

	errorCh := make(chan error)
	go mux.Run(errorCh)

	mux.Cancel()

	if mux.running == true {
		t.Error("Stopped mux still marked as running.")
	}
}

func TestMuxWritesToRings(t *testing.T) {
	mux := NewMux()
	errorCh := make(chan error)

	rings := make([]*ringbuf.Ringbuf, 2)
	rings[0] = ringbuf.NewRingbuf(10)
	rings[1] = ringbuf.NewRingbuf(10)

	go rings[0].Run()
	go rings[1].Run()
	go mux.Run(errorCh)

	mux.Add(rings[0])

	mux.Write("test0")
	mux.Write("test1")

	mux.Add(rings[1])

	mux.Write("test2")
	mux.Write("test3")

	mux.Cancel()

	readers := make([]*ringbuf.Reader, 2)
	readers[0] = ringbuf.NewReader(rings[0])
	readers[1] = ringbuf.NewReader(rings[1])

	dataCh := readers[0].ReadCh()

	if data := <-dataCh; data != "test0" {
		t.Error(fmt.Sprintf("Expected 'test0', got '%s'", data))
	}

	if data := <-dataCh; data != "test1" {
		t.Error(fmt.Sprintf("Expected 'test1', got '%s'", data))
	}

	if data := <-dataCh; data != "test2" {
		t.Error(fmt.Sprintf("Expected 'test2', got '%s'", data))
	}

	if data := <-dataCh; data != "test3" {
		t.Error(fmt.Sprintf("Expected 'test3', got '%s'", data))
	}

	rings[0].Cancel()

	dataCh = readers[1].ReadCh()

	if data := <-dataCh; data != "test2" {
		t.Error(fmt.Sprintf("Expected 'test2', got '%s'", data))
	}

	if data := <-dataCh; data != "test3" {
		t.Error(fmt.Sprintf("Expected 'test3', got '%s'", data))
	}

	rings[1].Cancel()
}
