package blocksync

import "io"

type CountWriter struct {
	io.Writer
	count int
}

func NewCountWriter(w io.Writer) *CountWriter {
	return &CountWriter{
		Writer: w,
		count:  0,
	}
}

func (w *CountWriter) Write(buf []byte) (int, error) {
	n, err := w.Writer.Write(buf)
	w.count += n
	return n, err
}

func (w *CountWriter) Count() int {
	return w.count
}
