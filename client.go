package black

import "os"

type client struct{}

// Run TODO
func (cli *client) Run() error {
	return errNotImplemtented
}

// Close TODO
func (cli *client) Close() error {
	return errNotImplemtented
}

// NewClient TODO
func NewClient(client string, file *os.File) (RunCloser, error) {
	return nil, errNotImplemtented
}
