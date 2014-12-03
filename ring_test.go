package ringbuf

import (
	"fmt"
	"testing"
)

func TestSimpleWriteRead(t *testing.T) {
	ring := NewRingbuf(3)
	reader := NewReader(ring)

	ring.write("test0")
	ring.write("test1")

	if val, ok := reader.read(); !ok || val != "test0" {
		t.Error(fmt.Sprintf("Expected value test0, got '%s'", val))
	}

	if val, ok := reader.read(); !ok || val != "test1" {
		t.Error(fmt.Sprintf("Expected value test1, got '%s'", val))
	}

	if val, ok := reader.read(); ok || val != nil {
		t.Error("Expected read fail")
	}

	ring.write("test2")

	if val, ok := reader.read(); !ok || val != "test2" {
		t.Error(fmt.Sprintf("Expected value test2, got '%s'", val))
	}

	ring.write("test3")
	ring.write("test4")

	if val, ok := reader.read(); !ok || val != "test3" {
		t.Error(fmt.Sprintf("Expected value test3, got '%s'", val))
	}

	if val, ok := reader.read(); !ok || val != "test4" {
		t.Error(fmt.Sprintf("Expected value test4, got '%s'", val))
	}

	ring.write("test5")
	ring.write("test6")
	ring.write("test7")

	if val, ok := reader.read(); !ok || val != "test5" {
		t.Error(fmt.Sprintf("Expected value test5, got '%s'", val))
	}

	if val, ok := reader.read(); !ok || val != "test6" {
		t.Error(fmt.Sprintf("Expected value test6, got '%s'", val))
	}
}

func TestWriteReadSlow(t *testing.T) {
	ring := NewRingbuf(3)
	reader := NewReader(ring)

	ring.write("test1")
	ring.write("test2")
	ring.write("test3")

	if val, ok := reader.read(); !ok || val != "test1" {
		t.Error(fmt.Sprintf("Expected value test1, got '%s'", val))
	}

	ring.write("test4")

	if val, ok := reader.read(); !ok || val != "test2" {
		t.Error(fmt.Sprintf("Expected value test2, got '%s'", val))
	}
}
