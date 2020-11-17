package blocksync

import "io"

type CountReader struct {
	io.Reader
	count int
}

func NewCountReader(r io.Reader) *CountReader {
	return &CountReader{
		Reader: r,
		count:  0,
	}
}

func (r *CountReader) Read(buf []byte) (int, error) {
	n, err := r.Reader.Read(buf)
	r.count += n
	return n, err
}

func (r *CountReader) Count() int {
	return r.count
}
