package black

import "os"

type proxy struct{}

// Run TODO
func (prx *proxy) Run() error {
	return errNotImplemtented
}

// Close TODO
func (prx *proxy) Close() error {
	return errNotImplemtented
}

// NewProxy TODO
func NewProxy(proxy, addr string, file *os.File) (RunCloser, error) {
	return nil, errNotImplemtented
}
