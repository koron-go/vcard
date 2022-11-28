/*
Package errio provides io.Reader with some errors.
*/
package errio

import (
	"errors"
	"io"
)

type Reader struct {
	r   io.Reader
	eof error
}

var _ io.Reader = (*Reader)(nil)

func NewReader(r io.Reader, eof error) *Reader {
	return &Reader{r: r, eof: eof}
}

func (r *Reader) Read(b []byte) (int, error) {
	n, err := r.r.Read(b)
	if errors.Is(err, io.EOF) && r.eof != nil {
		err = r.eof
	}
	return n, err
}
