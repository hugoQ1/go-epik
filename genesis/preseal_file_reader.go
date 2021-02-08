package genesis

import (
	"bytes"
	"crypto/rand"
	"io"
)

type contentReader struct {
	contentLength int
	contentRead   int

	zeroPadding bool
}

func (r *contentReader) Read(buf []byte) (n int, err error) {
	available := r.contentLength - r.contentRead

	if len(buf) <= available {
		n = copy(buf, PresealFile[r.contentRead:])
		r.contentRead += n
		return
	}

	if available > 0 {
		copy(buf[:available], PresealFile[r.contentRead:])
		r.contentRead = r.contentLength
	}

	// zero padding
	if r.zeroPadding {
		copy(buf[available:], bytes.Repeat([]byte{0}, len(buf)-available))
		return len(buf), nil
	}
	// rand padding
	nr, err := rand.Reader.Read(buf[available:])
	return available + nr, err
}

func NewZeroPaddingPresealFileReader() io.Reader {
	return &contentReader{
		contentLength: len(PresealFile),
		zeroPadding:   true,
	}
}

func NewRandPaddingPresealFileReader() io.Reader {
	return &contentReader{contentLength: len(PresealFile)}
}
