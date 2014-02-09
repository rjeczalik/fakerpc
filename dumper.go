package black

import "io"

// TODO
type dumper struct {
}

func newDumper(w io.WriteCloser) *dumper {
	return &dumper{}
}
