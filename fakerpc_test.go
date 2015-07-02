package fakerpc

import (
	"bytes"
	"net"
	"net/http"
	"net/url"
	"testing"
)

var (
	cli = [...]net.TCPAddr{{
		IP:   net.IPv4(192, 168, 14, 186),
		Port: 46793,
	}, {
		IP:   net.IPv4(192, 168, 14, 186),
		Port: 46794,
	}, {
		IP:   net.IPv4(192, 168, 14, 186),
		Port: 46795,
	}}
	srv = &net.TCPAddr{
		IP:   net.IPv4(192, 168, 16, 50),
		Port: 80,
	}
)

var log = &Log{T: []Transmission{{
	Src: &cli[0], Dst: srv,
	Raw: []byte("POST /1 HTTP/1.1\r\n\r\nHAI"),
}, {
	Src: srv, Dst: &cli[0],
	Raw: []byte("HTTP/1.1 200 OK\nContent-Length: 4\r\n\r\nHAAI"),
}, {
	Src: &cli[0], Dst: srv,
	Raw: []byte("POST /2 HTTP/1.1\nContent-Length: 4\nConnection: close\r\n\r\nBAAI"),
}, {
	Src: srv, Dst: &cli[0],
	Raw: []byte("HTTP/1.1 200 OK\nContent-Length: 5\r\n\r\nBAAAI"),
}, {
	Src: &cli[1], Dst: srv,
	Raw: []byte("POST /3 HTTP/1.1\nConnection: close\r\n\r\nHAI"),
}, {
	Src: srv, Dst: &cli[1],
	Raw: []byte("HTTP/1.1 200 OK\r\n\r\n"),
}, {
	Src: &cli[2], Dst: srv,
	Raw: []byte("POST /4 HTTP/1.1\nContent-Length: 4\r\n\r\nHAAICho3wama"),
}, {
	Src: srv, Dst: &cli[2],
	Raw: []byte("HTTP/1.1 200 OK\nContent-Length: 5\r\n\r\nHAAAI"),
}, {
	Src: &cli[2], Dst: srv,
	Raw: []byte("POST /5 HTTP/1.1\nContent-Length: 6\nConnection: close\r\n\r\nBAAAAIEichee6e"),
}, {
	Src: srv, Dst: &cli[2],
	Raw: []byte("HTTP/1.1 200 OK\nContent-Length: 7\r\n\r\nBAIBAAI"),
}}}

var expconn = [][]Connection{{{
	Req:     &http.Request{Method: "POST", ContentLength: 3, URL: &url.URL{Path: "/1"}},
	ReqBody: []byte("HAI"), Res: []byte("HTTP/1.1 200 OK\nContent-Length: 4\r\n\r\nHAAI"),
}, {
	Req:     &http.Request{Method: "POST", ContentLength: 4, URL: &url.URL{Path: "/2"}, Close: true},
	ReqBody: []byte("BAAI"), Res: []byte("HTTP/1.1 200 OK\nContent-Length: 5\r\n\r\nBAAAI"),
}}, {{
	Req:     &http.Request{Method: "POST", ContentLength: 3, URL: &url.URL{Path: "/3"}, Close: true},
	ReqBody: []byte("HAI"), Res: []byte("HTTP/1.1 200 OK\r\n\r\n"),
}}, {{
	Req:     &http.Request{Method: "POST", ContentLength: 4, URL: &url.URL{Path: "/4"}},
	ReqBody: []byte("HAAI"), Res: []byte("HTTP/1.1 200 OK\nContent-Length: 5\r\n\r\nHAAAI"),
}, {
	Req:     &http.Request{Method: "POST", ContentLength: 6, URL: &url.URL{Path: "/5"}, Close: true},
	ReqBody: []byte("BAAAAI"), Res: []byte("HTTP/1.1 200 OK\nContent-Length: 7\r\n\r\nBAIBAAI"),
}}}

func TestNewConnections(t *testing.T) {
	conn, err := NewConnections(log)
	if err != nil {
		t.Fatalf("expected err=nil; got %q", err)
	}
	if len(conn) != len(expconn) {
		t.Fatalf("expected len(conn)=%d; got %d", len(expconn), len(conn))
	}
	for i, conn := range conn {
		if len(conn) != len(expconn[i]) {
			t.Errorf("expected len(conn[%d])=%d; got %d", i, len(expconn[i]), len(conn))
			continue
		}
		for j, conn := range conn {
			if conn.Req.Method != expconn[i][j].Req.Method {
				t.Errorf("expected conn[%d][%d].Req.Method=%q; got %q", i, j,
					expconn[i][j].Req.Method, conn.Req.Method)
			}
			if conn.Req.ContentLength != expconn[i][j].Req.ContentLength {
				t.Errorf("expected conn[%d][%d].Req.ContentLength=%d; got %d", i, j,
					expconn[i][j].Req.ContentLength, conn.Req.ContentLength)
			}
			if conn.Req.URL.Path != expconn[i][j].Req.URL.Path {
				t.Errorf("expected conn[%d][%d].Req.URL.Path=%q; got %q", i, j,
					expconn[i][j].Req.URL.Path, conn.Req.URL.Path)
			}
			if conn.Req.Close != expconn[i][j].Req.Close {
				t.Errorf("expected conn[%d][%d].Req.Close=%v; got %v", i, j,
					expconn[i][j].Req.Close, conn.Req.Close)
			}
			if !bytes.Equal(conn.ReqBody, expconn[i][j].ReqBody) {
				t.Errorf("expected conn[%d][%d].ReqBody=%q; got %q", i, j,
					expconn[i][j].ReqBody, conn.ReqBody)
			}
			if !bytes.Equal(conn.Res, expconn[i][j].Res) {
				t.Errorf("expected conn[%d][%d].Res=%q; got %q", i, j,
					expconn[i][j].Res, conn.Res)
			}
		}
	}
}

func TestNewConnectionsErr(t *testing.T) {
	log := []*Log{
		nil,
		{},
		{T: []Transmission{{Src: &cli[0], Dst: srv, Raw: []byte{}}}},
		{T: []Transmission{{Src: &cli[0], Dst: srv, Raw: []byte("Ic0aethu")}}},
		{T: []Transmission{{Src: &cli[0], Dst: srv, Raw: []byte("HTTP/1.1 200 OK\nContent-Length: 4\r\n\r\nX")}}},
	}
	for i, log := range log {
		if _, err := NewConnections(log); err == nil {
			t.Errorf("expected err!=nil (log[%d])", i)
		}
	}
}
