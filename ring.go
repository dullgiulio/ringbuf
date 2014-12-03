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

func (r *Reader) read() (interface{}, bool) {
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
