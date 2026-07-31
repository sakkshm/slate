package runner

import "bytes"

type lineWriter struct {
	buf     bytes.Buffer
	pending []byte
	sink    func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {

	w.buf.Write(p)

	start := 0
	for i, ch := range p {
		// append till linebreak, then sink
		if ch == '\n' {
			w.pending = append(w.pending, p[start:i]...)

			if w.sink != nil {
				w.sink(string(w.pending))
			}

			// clear arr while leaving capacity
			w.pending = w.pending[:0]
			start = i + 1
		}
	}

	// keep pending unfinished lines for next iteration
	w.pending = append(w.pending, p[start:]...)
	return len(p), nil
}
