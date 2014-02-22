package black

import "errors"

// TODO
var errNotImplemtented = errors.New("bx: not implemented")

// RunCloser TODO
type RunCloser interface {
	Run() error
	Close() error
}
