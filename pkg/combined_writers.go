package pkg

import (
	"io"

	"go.uber.org/multierr"
)

type CombinedWriter struct {
	Writers []io.Writer
	Err     error
}

func NewCombinedWriter(writers ...io.Writer) *CombinedWriter {
	cw := &CombinedWriter{}
	for _, w := range writers {
		cw.Writers = append(cw.Writers, w)
	}
	return cw
}

func (cw CombinedWriter) Write(p []byte) (n int, err error) {
	n = 0
	for _, w := range cw.Writers {
		written, werr := w.Write(p)
		if werr != nil {
			err = multierr.Combine(err, werr)
			continue
		}
		n += written
	}
	return n, err
}
