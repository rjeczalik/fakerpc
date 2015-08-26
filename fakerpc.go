package fakerpc

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// ErrAlreadyRunning is returned when calling ListenAndServe on a server which
// was already started by ListenAndServe.
var ErrAlreadyRunning = errors.New("fakerpc: server is already running")

// ErrNotRunning is returned when calling Stop on a server which wasn't started
// by ListenAndServe or was already stopped by Stop.
var ErrNotRunning = errors.New("fakerpc: server is not running")

// A Transmission represents a single raw data transmission between two TCP
// end points.
type Transmission struct {
	// Time of the start of the transmission.
	Time time.Time

	// Src is a TCP address of the source.
	Src *net.TCPAddr

	// Dst is a TCP address of the destination.
	Dst *net.TCPAddr

	// Raw contains all the recorded bytes sent from Src to Dst until Dst began
	// replying back to Src.
	Raw []byte
}

// A Log represents communication session, either captured by a Proxy or parsed
// from a ngrep output.
type Log struct {
	// Network is an address of the networks, in which communication took place.
	Networks []*net.IPNet

	// Filter is an effective pcap filter applied to the recording session.
	Filter string

	// T holds captured transmissions.
	T []Transmission

	// Coalesce is a maximum interval, within wihch all the writes/reads will get
	// coalesced into single transmission.
	Coalesce time.Duration
}

// Net returns the network names with the mask printed in a IP form instead of
// hexadecimal one.
func (l *Log) Net() []string {
	s := make([]string, 0, len(l.Networks))
	for _, network := range l.Networks {
		s = append(s, fmt.Sprintf("%v/%v", ipnil(network.IP), masktoip(network.Mask)))
	}
	return s
}

// NewLog gives a new Log.
func NewLog() *Log {
	return &Log{T: make([]Transmission, 0)}
}

// ReadLog gives Log decoded from the given file. It assumes the file contains
// gzipped, gob-encoded Log struct. If it does not, it treats the file as a ngrep
// output.
func ReadLog(file string) (*Log, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	l := NewLog()
	r, err := gzip.NewReader(f)
	if err != nil {
		f.Seek(0, 0)
		return l, NgrepUnmarshal(f, l)
	}
	defer r.Close()
	if err = gob.NewDecoder(r).Decode(l); err != nil {
		f.Seek(0, 0)
		return l, NgrepUnmarshal(f, l)
	}
	return l, nil
}

// WriteLog writes gzipped, gob-encoded Log struct to the file.
func WriteLog(file string, l *Log) error {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}
	defer w.Close()
	return gob.NewEncoder(w).Encode(l)
}

// A Connection represents a single request/reponse communication.
type Connection struct {
	Req     *http.Request // a HTTP header of the request
	ReqBody []byte        // a body of the request
	Res     []byte        // raw response
}

// Connections represent Log's transmissions grouped per connection.
type Connections [][]Connection

// NewConnections gives a Connections for the given log.
func NewConnections(log *Log) (Connections, error) {
	if log == nil || len(log.T) == 0 {
		return nil, errors.New("fakerpc: log is either nil or empty")
	}
	c, index := make(Connections, 0), make(map[string]int)
	for i := 0; i < len(log.T); {
		addr := log.T[i].Src.String()
		n, ok := index[addr]
		if !ok {
			c = append(c, make([]Connection, 0))
			n = len(c) - 1
			index[addr] = n
		}
		header, body := SplitHeaderBody(log.T[i].Raw)
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(header)))
		if err != nil {
			return nil, err
		}
		var conn = Connection{Req: req}
		if req.ContentLength == 0 {
			req.ContentLength = int64(len(body))
		}
		if int64(len(body)) < req.ContentLength {
			return nil, errors.New("fakerpc: recorded body length is too small")
		}
		body = body[:req.ContentLength]
		if len(body) > 0 {
			conn.ReqBody = make([]byte, len(body))
			copy(conn.ReqBody, body)
		}
		if i+1 < len(log.T) && tcpaddrequal(log.T[i].Src, log.T[i+1].Dst) {
			i += 1
			conn.Res = make([]byte, len(log.T[i].Raw))
			copy(conn.Res, log.T[i].Raw)
		}
		i += 1
		c[n] = append(c[n], conn)
	}
	return c, nil
}

// SplitHeaderBody splits raw HTTP request/response into header and body.
func SplitHeaderBody(p []byte) (header []byte, body []byte) {
	if n := bytes.Index(p, []byte("\r\n\r\n")); n != -1 {
		header = p[:n+4]
		body = p[n+4:]
	}
	return
}
