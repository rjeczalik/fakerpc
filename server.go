package black

import "os"

type server struct{}

// Run TODO
func (srv *server) Run() error {
	return errNotImplemtented
}

// Close TODO
func (srv *server) Close() error {
	return errNotImplemtented
}

// NewServer TODO
func NewServer(server string, file *os.File) (RunCloser, error) {
	return nil, errNotImplemtented
}
