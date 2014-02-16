package black

import (
	"bytes"
	"io"
)

// ListeningBuffer is a bytes.Buffer that implements io.Closer as a callback.
type ClosingBuffer struct {
	bytes.Buffer
	OnClose func()
}

// Close invokes onclose callback.
func (cb *ClosingBuffer) Close() (err error) {
	if cb.OnClose != nil {
		cb.OnClose()
	}
	return
}

type multiCloser struct {
	closers []io.Closer
}

func (mc *multiCloser) Close() (err error) {
	var e error
	for _, c := range mc.closers {
		e = c.Close()
		if err == nil {
			err = e
		}
	}
	return
}

// MultiCloser returns a Closer that's the logical concatenation of
// the provided closers. They're being closed sequentialy. All closers are
// guaranteed to close. In case of an error first encountered error is returned.
func MultiCloser(closers ...io.Closer) io.Closer {
	return &multiCloser{closers}
}

type teeReadCloser struct {
	io.Reader
	io.Closer
}

// TeeReadCloser is similar to io.TeeReader with an addition that if w implements
// io.Closer it will close after r does.
func TeeReadCloser(r io.ReadCloser, w io.Writer) io.ReadCloser {
	if w, ok := w.(io.WriteCloser); ok {
		return &teeReadCloser{io.TeeReader(r, w), MultiCloser(r, w)}
	}
	return &teeReadCloser{io.TeeReader(r, w), r}
}
