package black

import (
	"encoding/gob"
	"log"
	"os"
)

type client struct {
	file *os.File
}

// Run TODO
func (cli *client) Run() error {
	session := make(Session, 0)
	file := &SessionFile{new(Header), &session}
	dec := gob.NewDecoder(cli.file)
	if err := dec.Decode(file); err != nil {
		return err
	}
	log.Println("[DEBUG] Version =", file.Version)
	log.Println("[DEBUG] Schema =", file.Scheme)
	log.Println("[DEBUG] Target =", file.Target)
	for i, c := range session {
		log.Printf("[DEBUG] len(session[%d]) = %d", i, len(c))
		for i, m := range c {
			log.Printf("[DEBUG] [%d] session[%d].Raw.Len() = %d", m.Type, i, m.Raw.Len())
		}
	}
	return nil
}

// Close TODO
func (cli *client) Close() error {
	return errNotImplemtented
}

// NewClient TODO
func NewClient(cli string, file *os.File) (RunCloser, error) {
	return &client{file}, nil
}
