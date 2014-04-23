package fakerpc

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

var errNoResponse = errors.New("fakerpc: no response recorded for the request")

// Server TODO(rjeczalik): document
type Server struct {
	// TODO(rjeczalik): synchronized setter?
	Reply func(src, dst *net.TCPAddr, bodyLen int64, err error)
	m     sync.Mutex
	conn  Connections
	l     net.Listener
	src   *net.TCPAddr
	addr  string
	isrun uint32
	count int
	wg    sync.WaitGroup
}

func tcpaddrnil(addr net.Addr) (tcpa *net.TCPAddr) {
	if a, err := tcpaddr(addr); err == nil {
		tcpa = a
	}
	return
}

func write500(rw net.Conn, err error) {
	s := err.Error()
	io.WriteString(rw, fmt.Sprintf("HTTP/1.1 500 Internal Server Error\r\n"+
		"Content-Length: %d\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		len(s), s))
}

// ServeConn TODO(rjeczalik): document
func (srv *Server) ServeConn(rw net.Conn, c []Connection) {
	var (
		n   int64
		err error
		req *http.Request
		r   = bufio.NewReader(rw)
		rem = tcpaddrnil(rw.RemoteAddr())
	)
	for i := 0; ; i++ {
		if req, err = http.ReadRequest(r); err != nil {
			break
		}
		n, err = io.Copy(ioutil.Discard, req.Body)
		req.Body.Close()
		if i >= len(c) {
			write500(rw, errNoResponse)
			srv.Reply(rem, srv.src, n, errNoResponse)
			continue
		}
		srv.Reply(rem, srv.src, n, err)
		if err != nil {
			write500(rw, err)
			continue
		}
		if c[i].Res != nil {
			_, err = io.Copy(rw, bytes.NewBuffer(c[i].Res))
			srv.Reply(srv.src, rem, int64(len(c[i].Res)), err)
		}
	}
	if err != io.EOF {
		srv.Reply(rem, srv.src, 0, err)
	}
	rw.Close()
	srv.wg.Done()
}

var noopReply = func(*net.TCPAddr, *net.TCPAddr, int64, error) {}

// NewServer TODO(rjeczalik): document
func NewServer(addr string, log *Log) (srv *Server, err error) {
	srv = &Server{Reply: noopReply, addr: addr}
	srv.conn, err = NewConnections(log)
	return
}

// ListenAndServe TODO(rjeczalik): document
func (srv *Server) ListenAndServe() (err error) {
	if atomic.CompareAndSwapUint32(&srv.isrun, 0, 1) {
		defer func() {
			// Ignore "use of closed network connection" comming from closed
			// net.Listener when p was explicitely stopped.
			if !atomic.CompareAndSwapUint32(&srv.isrun, 1, 0) {
				err = nil
			}
		}()
		srv.m.Lock()
		if srv.l, err = net.Listen("tcp", srv.addr); err != nil {
			srv.m.Unlock()
			return
		}
		if srv.src, err = tcpaddr(srv.l.Addr()); err != nil {
			srv.m.Unlock()
			return
		}
		srv.m.Unlock()
		var (
			c    []Connection
			conn net.Conn
		)
		for {
			if conn, err = srv.l.Accept(); err != nil {
				return
			}
			c = srv.conn[srv.count]
			srv.count += 1
			srv.wg.Add(1)
			go srv.ServeConn(conn, c)
			if srv.count == len(srv.conn) {
				srv.Stop()
				break
			}
		}
		srv.wg.Wait()
		return
	}
	return ErrAlreadyRunning
}

// Running TODO(rjeczalik): document
func (srv *Server) Running() bool {
	return atomic.LoadUint32(&srv.isrun) == 1
}

// Stop TODO(rjeczalik): document
func (srv *Server) Stop() (err error) {
	err = ErrNotRunning
	if atomic.CompareAndSwapUint32(&srv.isrun, 1, 0) {
		srv.m.Lock()
		err = srv.l.Close()
		srv.m.Unlock()
	}
	return
}

// Addr TODO(rjeczalik): document
func (srv *Server) Addr() (addr net.Addr) {
	if atomic.LoadUint32(&srv.isrun) == 1 {
		srv.m.Lock()
		addr = srv.l.Addr()
		srv.m.Unlock()
	}
	return
}
