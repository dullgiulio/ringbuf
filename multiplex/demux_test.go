package multiplex

import (
	"bitbucket.org/dullgiulio/ringbuf"
	"fmt"
	"sync"
	"testing"
)

func TestDemuxCancel(t *testing.T) {
	demux := NewDemux()
	errorCh := make(chan error)

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

	r := NewDemuxReader(ringbuf.NewRingbufReader(ring))
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
	r := NewDemuxReader(ringbuf.NewRingbufReader(rings[0]))
	r.SetOnCancel(func() {
		wg.Done()
	})
	demux.Add(r)

	rings[0].Write("test0-0")
	rings[0].Write("test0-1")
	rings[0].Write("test0-2")

	wg.Add(1)
	r = NewDemuxReader(ringbuf.NewRingbufReader(rings[1]))
	r.SetOnCancel(func() {
		wg.Done()
	})
	demux.Add(r)

	rings[1].Write("test1-0")
	rings[1].Write("test1-1")
	rings[1].Write("test1-2")

	// Add third ringbuf to demux
	wg.Add(1)
	r = NewDemuxReader(ringbuf.NewRingbufReader(rings[2]))
	r.SetOnCancel(func() {
		wg.Done()
	})
	demux.Add(r)

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

	demux.Cancel()

	<-finishCh
	wg.Wait()
}
