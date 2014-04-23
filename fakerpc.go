package fakerpc

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
)

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
	c, index := make(Connections, 0), make(map[string]int)
	for i := 0; i < len(log.T); {
		addr := log.T[i].Src.String()
		n, ok := index[addr]
		if !ok {
			c = append(c, make([]Connection, 0))
			n = len(c) - 1
			index[addr] = n
		}
		header, body := SplitHTTP(log.T[i].Raw)
		req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(header)))
		if err != nil {
			return nil, err
		}
		var conn = Connection{Req: req}
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

// SplitHTTP TODO(rjeczalik): document
func SplitHTTP(p []byte) (header []byte, body []byte) {
	if n := bytes.Index(p, []byte("\r\n\r\n")); n != -1 {
		header = p[:n+4]
		body = p[n+4:]
	}
	return
}
