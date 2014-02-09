package black

import (
	"bytes"
	"net/http"
)

type round struct {
	request struct {
		timestamp int64
		body      bytes.Buffer
		*http.Request
	}
	response struct {
		timestamp int64
		body      bytes.Buffer
		*http.Response
	}
}

// TODO
type recordingTransport struct {
	http.Transport
	d *dumper
}

func newRecordingTransport(d *dumper) *recordingTransport {
	return &recordingTransport{
		http.Transport{Proxy: http.ProxyFromEnvironment},
		d,
	}
}
