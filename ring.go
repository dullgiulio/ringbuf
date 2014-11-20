package ringbuf

// Unsafe write. Must be called by IO main loop.
func (r *Ringbuf) write(data interface{}) {
	if r.pos >= r.size {
		r.pos = 0
		r.cycles++
	}

	r.data[r.pos] = data
	r.pos++
}

func (r *Ringbuf) writeOrStarve(data interface{}, lastReader *RingbufReader) bool {
	if r.pos >= r.size {
		r.pos = 0
		r.cycles++
	}

	if lastReader.pos == r.pos &&
		lastReader.cycles < r.cycles {
		return false
	}

	r.data[r.pos] = data
	r.pos++
	return true
}

// TODO: instead of bool, pass a status: done, doneWithSkip, notDone
func (r *RingbufReader) read() (interface{}, bool) {
	// 1. All normal. We are behind the writer
	if r.pos <= r.ring.pos-1 && r.cycles == r.ring.cycles {
		data := r.ring.data[r.pos]

		r.pos++

		if r.pos >= r.ring.size {
			r.pos = 0
			r.cycles++
		}

		return data, true
	}

	// 2. We are waiting for the writer
	if r.pos >= r.ring.pos-1 && r.cycles >= r.ring.cycles {
		return "", false
	}

	// 3. We are too far behind, the writer has wrapped around
	if r.cycles < r.ring.cycles {
		diffCycles := r.ring.cycles - r.cycles

		// Distance between reader and writer > one ring
		if r.ring.size-r.pos+r.ring.pos*diffCycles >= r.ring.size {
			// Skip to where the ring starts now.
			// TODO: I might want to get a warning here.
			r.cycles = r.ring.cycles
			r.pos = 0
			return r.ring.data[0], true
		} else {
			data := r.ring.data[r.pos]

			r.pos++

			if r.pos >= r.ring.size {
				r.pos = 1
				r.cycles++
			}

			return data, true
		}
	}

	return "", false
}

func SlowestReader(readers []*RingbufReader) (slowestReader *RingbufReader) {
	for _, r := range readers {
		if slowestReader == nil {
			slowestReader = r
			continue
		}

		if slowestReader.cycles >= r.cycles &&
			slowestReader.pos > r.cycles {
			slowestReader = r
		}
	}

	return
}