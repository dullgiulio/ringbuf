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
		return nil, false
	}

	// 3. We are too far behind, the writer has wrapped around
	if r.cycles < r.ring.cycles {
		if r.cycles == r.ring.cycles-1 && r.pos >= r.ring.pos {
			// Can continue normally, the writer is still behind us
			data := r.ring.data[r.pos]

			r.pos++

			if r.pos >= r.ring.size {
				r.pos = 0
				r.cycles++
			}

			return data, true
		}

		// Two or more cycles behind, cannot rescue this
		// Instead, skip to where the ring starts now
		r.cycles = r.ring.cycles
		r.pos = 1
		return r.ring.data[0], false
	}

	return nil, false
}
