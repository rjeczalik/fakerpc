package black

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestClosingBuffer(t *testing.T) {
	closed := false
	if err := NewClosingBuffer(func() { closed = true }).Close(); err != nil {
		t.Errorf("expected err to be nil, was %q instead", err)
	}
	if !closed {
		t.Error("expected closed to be true")
	}
}

type dummyCloser struct{ err error }

func (dc *dummyCloser) Close() error { return dc.err }

func TestMultiCloser(t *testing.T) {
	closed, errClose := []bool{false, false}, errors.New("close")
	closers := []io.Closer{
		NewClosingBuffer(func() { closed[0] = true }),
		&dummyCloser{errClose},
		NewClosingBuffer(func() { closed[1] = true }),
	}
	if err := MultiCloser(closers...).Close(); err == nil || err != errClose {
		t.Errorf("expected err to be %q, was %q instead", errClose, err)
	}
	for i, c := range closed {
		if !c {
			t.Errorf("expected closed[%d] to be true", i)
		}
	}
}

func TestTeeReadCloser(t *testing.T) {
	closed, src, errClose := false, []byte("hello world"), errors.New("close")
	rb, wb := NewClosingBuffer(func() { closed = true }), new(bytes.Buffer)
	rb.Write(src)
	dst := make([]byte, len(src))
	r := TeeReadCloser(rb, wb)
	if n, err := io.ReadFull(r, dst); err != nil || n != len(src) {
		t.Errorf("expected io.ReadFull(r, dst) = %d, nil; was %d, %q instead",
			len(src), n, err)
	}
	if !bytes.Equal(dst, src) {
		t.Errorf("expected dst to be %q, was %q instead", src, dst)
	}
	if !bytes.Equal(wb.Bytes(), src) {
		t.Errorf("expected wb to be %q, was %q instead", src, wb.Bytes())
	}
	if err := rb.Close(); err != nil || !closed {
		t.Errorf("expected err, closed = nil, true; was %q, %t instead", err, closed)
	}
	closed = false
	w := &struct {
		io.Writer
		io.Closer
	}{nil, &dummyCloser{errClose}}
	if err := TeeReadCloser(rb, w).Close(); err != errClose || !closed {
		t.Errorf("expected err, closed = %q, true; was %q, %t instead", errClose,
			err, closed)
	}
}
