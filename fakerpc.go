package fakerpc

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
)

// ErrAlreadyRunning TODO(rjeczalik): document
var ErrAlreadyRunning = errors.New("fakerpc: server is already running")

// ErrNotRunning TODO(rjeczalik): document
var ErrNotRunning = errors.New("fakerpc: server is not running")

// Transmission TODO(rjeczalik): document
type Transmission struct {
	Src *net.TCPAddr
	Dst *net.TCPAddr
	Raw []byte
}

// Log TODO(rjeczalik): document
type Log struct {
	Network net.IPNet
	Filter  string
	T       []Transmission
}

// NetIP TODO(rjeczalik): document
func (l *Log) NetIP() net.IP {
	return ipnil(l.Network.IP)
}

// NetMask TODO(rjeczalik): document
func (l *Log) NetMask() net.IP {
	return masktoip(l.Network.Mask)
}

// NetFilter TODO(rjeczalik): document
func (l *Log) NetFilter() string {
	if l.Filter != "" {
		return l.Filter
	}
	if len(l.T) == 0 {
		return "(none)"
	}
	return fmt.Sprintf("(ip or ipv6) and ( host %s and port %d )",
		l.T[0].Dst.IP, l.T[0].Dst.Port)
}

// NewLog TODO(rjeczalik): document
func NewLog() *Log {
	return &Log{T: make([]Transmission, 0)}
}

// ReadLog TODO(rjeczalik): document
func ReadLog(file string) (*Log, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	l := NewLog()
	r, err := gzip.NewReader(f)
	if err != nil {
		return l, NgrepUnmarshal(f, l)
	}
	defer r.Close()
	var buf bytes.Buffer
	if err = gob.NewDecoder(io.TeeReader(r, &buf)).Decode(l); err == nil {
		return l, nil
	}
	return l, NgrepUnmarshal(bytes.NewBuffer(buf.Bytes()), l)
}

// WriteLog TODO(rjeczalik): document
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

// Connection TODO(rjeczalik): document
type Connection struct {
	Req     *http.Request
	ReqBody []byte
	Res     []byte
}

// Connections TODO(rjeczalik): document
type Connections [][]Connection

func equal(lhs, rhs *net.TCPAddr) bool {
	return lhs == rhs || (lhs.IP.Equal(rhs.IP) && lhs.Port == rhs.Port)
}

// NewConnections TODO(rjeczalik): document
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
		if i+1 < len(log.T) && equal(log.T[i].Src, log.T[i+1].Dst) {
			i += 1
			conn.Res = make([]byte, len(log.T[i].Raw))
			copy(conn.Res, log.T[i].Raw)
		}
		i += 1
		c[n] = append(c[n], conn)
	}
	return c, nil
}

// SplitHeaderBody TODO(rjeczalik): document
func SplitHeaderBody(p []byte) (header []byte, body []byte) {
	if n := bytes.Index(p, []byte("\r\n\r\n")); n != -1 {
		header = p[:n+4]
		body = p[n+4:]
	}
	return
}
