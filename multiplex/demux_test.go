package multiplex

import (
	"fmt"
	"github.com/dullgiulio/ringbuf"
	"sync"
	"testing"
)

func TestDemuxCancel(t *testing.T) {
	demux := NewDemux()
	errorCh := make(chan error)

    if demux.String() == "" {
        t.Error("Expected a string coversion")
    }

	go demux.Run(errorCh)
	demux.Cancel()
}

func TestDemuxReadFromOne(t *testing.T) {
	ring := ringbuf.NewRingbuf(100)

	demux := NewDemux()
	errorCh := make(chan error)

	finishedCh := make(chan bool)
	go func() {
		demux.Run(errorCh)
	}()
	go ring.Run()

	r := NewDemuxReader(ringbuf.NewReader(ring))
	r.SetOnCancel(func() {
		finishedCh <- true
	})
	demux.Add(r)

	ring.Write("test0-0")
	ring.Write("test0-1")
	ring.Write("test0-2")

	ring.Cancel()

	reader := demux.Reader()
	readCh := reader.ReadCh()

	result := map[string]bool{
		"test0-0": false,
		"test0-1": false,
		"test0-2": false,
	}

	for i := 0; i < 3; i++ {
		raw := <-readCh
		data := raw.(string)

		if checked, ok := result[data]; ok {
			if checked {
				t.Error(fmt.Sprintf("Found '%s' twice in resulting ringbuf\n", data))
			} else {
				result[data] = true
			}
		} else {
			t.Error(fmt.Sprintf("Found '%s' which is not expected\n", data))
		}
	}

	for data, found := range result {
		if !found {
			t.Error(fmt.Sprintf("Not found expected '%s'\n", data))
		}
	}

	demux.Cancel()
	<-finishedCh
}

func TestDemuxReadFromMany(t *testing.T) {
	var wg sync.WaitGroup

	finishCh := make(chan bool)

	// Create and fill three ringbufs.
	rings := make([]*ringbuf.Ringbuf, 3)
	rings[0] = ringbuf.NewRingbuf(100)
	rings[1] = ringbuf.NewRingbuf(100)
	rings[2] = ringbuf.NewRingbuf(100)

	go rings[0].Run()
	go rings[1].Run()
	go rings[2].Run()

	// Add two ringbufs to demux
	demux := NewDemux()
	errorCh := make(chan error)

	go func() {
		demux.Run(errorCh)
		finishCh <- true
	}()

	wg.Add(1)
	r0 := NewDemuxReader(ringbuf.NewReader(rings[0]))
	r0.SetOnCancel(func() {
		wg.Done()
	})
	demux.Add(r0)
    demux.Add(r0)

    if err := <- errorCh; err == nil {
        t.Error("Expected error after inserting the same writer twice")
    }

	rings[0].Write("test0-0")
	rings[0].Write("test0-1")
	rings[0].Write("test0-2")

	wg.Add(1)
    r1 := NewDemuxReader(ringbuf.NewReader(rings[1]))
	r1.SetOnCancel(func() {
		wg.Done()
	})
	demux.Add(r1)

	rings[1].Write("test1-0")
	rings[1].Write("test1-1")
	rings[1].Write("test1-2")

	// Add third ringbuf to demux
	wg.Add(1)
    r2 := NewDemuxReader(ringbuf.NewReader(rings[2]))
	r2.SetOnCancel(func() {
		wg.Done()
	})
	demux.Add(r2)

	// Terminate ringbufs.
	rings[0].Cancel()
	rings[1].Cancel()

	rings[2].Write("test2-0")
	rings[2].Write("test2-1")
	rings[2].Write("test2-2")

	rings[2].Cancel()

	reader := demux.Reader()
	readCh := reader.ReadCh()

	result := map[string]bool{
		"test0-0": false,
		"test0-1": false,
		"test0-2": false,
		"test1-0": false,
		"test1-1": false,
		"test1-2": false,
		"test2-0": false,
		"test2-1": false,
		"test2-2": false,
	}

	for i := 0; i < 9; i++ {
		raw := <-readCh
		data := raw.(string)

		if checked, ok := result[data]; ok {
			if checked {
				t.Error(fmt.Sprintf("Found '%s' twice in resulting ringbuf\n", data))
			} else {
				result[data] = true
			}
		} else {
			t.Error(fmt.Sprintf("Found '%s' which is not expected\n", data))
		}
	}

	for data, found := range result {
		if !found {
			t.Error(fmt.Sprintf("Not found expected '%s'\n", data))
		}
	}

    demux.Remove(r0)
    demux.Remove(r1)
    demux.Remove(r2)
    demux.Remove(r2)

    if err := <- errorCh; err == nil {
        t.Error("Expected error after removing reader twice")
    }

    demux.Cancel()

	<-finishCh
	wg.Wait()
}
