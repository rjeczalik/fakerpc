package fakerpc

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

var noopRecord = func(*Transmission) {}

type recConn struct {
	net.Conn
	t      []Transmission
	commit func([]Transmission)
	rec    func(*Transmission)
	src    *net.TCPAddr
	dst    *net.TCPAddr
	wg     *sync.WaitGroup
}

func (rc *recConn) record(p []byte, src, dst *net.TCPAddr) {
	if rc.t[len(rc.t)-1].Src != src {
		rc.rec(&rc.t[len(rc.t)-1])
		rc.t = append(rc.t, Transmission{})
	}
	if len(p) != 0 {
		t := &rc.t[len(rc.t)-1]
		if t.Src == nil {
			t.Src, t.Dst = src, dst
		}
		t.Raw = append(t.Raw, p...)
	}
}

func (rc *recConn) Read(p []byte) (n int, err error) {
	n, err = rc.Conn.Read(p)
	rc.record(p[:n], rc.dst, rc.src)
	return
}

func (rc *recConn) Write(p []byte) (n int, err error) {
	n, err = rc.Conn.Write(p)
	rc.record(p[:n], rc.src, rc.dst)
	return
}

func (rc *recConn) Close() (err error) {
	err = rc.Conn.Close()
	if len(rc.t) > 0 && rc.t[len(rc.t)-1].Src == nil {
		rc.t = rc.t[:len(rc.t)-1]
	}
	rc.commit(rc.t)
	rc.wg.Done()
	return
}

type recListener struct {
	log Log
	wg  sync.WaitGroup
	m   sync.Mutex
	lis net.Listener
	src *net.TCPAddr
	dst *net.TCPAddr
	rec func(*Transmission)
}

func newRecListener(lis net.Listener, u *url.URL, rec func(*Transmission)) (l *recListener, err error) {
	src, err := urltotcpaddr(u)
	if err != nil {
		return
	}
	ipnet, err := ipnetaddr(lis.Addr())
	if err != nil {
		return
	}
	l = &recListener{
		log: Log{
			Network: net.IPNet{IP: ipnet.IP, Mask: ipnet.Mask},
			Filter:  fmt.Sprintf("(ip or ipv6) and ( host %s and port %d )", src.IP, src.Port),
			T:       make([]Transmission, 0),
		},
		lis: lis,
		src: src,
		rec: rec,
	}
	return
}

func (rl *recListener) Accept() (c net.Conn, err error) {
	if c, err = rl.lis.Accept(); err != nil {
		return
	}
	dst, err := tcpaddr(c.RemoteAddr())
	if err != nil {
		c.Close()
		return nil, err
	}
	c = &recConn{
		Conn: c,
		t: []Transmission{{
			Src: dst,
			Dst: rl.src,
			Raw: make([]byte, 0),
		}},
		commit: func(t []Transmission) {
			rl.m.Lock()
			rl.log.T = append(rl.log.T, t...)
			rl.m.Unlock()
		},
		src: rl.src,
		dst: dst,
		wg:  &rl.wg,
		rec: rl.rec,
	}
	rl.wg.Add(1)
	return
}

// TODO(rjeczalik): Forcefully close open connections?
func (rl *recListener) Close() (err error) {
	err = rl.lis.Close()
	rl.wg.Wait()
	return
}

func (rl *recListener) Addr() net.Addr {
	return rl.lis.Addr()
}

// A Proxy represents a single host HTTP reverse proxy which records all the
// transmission it handles.
type Proxy struct {
	// Record function is called after each transmission is successfully completed.
	Record func(*Transmission)
	m      sync.Mutex
	wgr    sync.WaitGroup
	targ   *url.URL
	rl     *recListener
	srv    *http.Server
	addr   string
	isrun  uint32
}

// NewProxy gives new Proxy for the given target URL and listening on the given
// TCP network address.
func NewProxy(addr, target string) (*Proxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		Record: noopRecord,
		targ:   u,
		addr:   addr,
		srv:    &http.Server{Handler: httputil.NewSingleHostReverseProxy(u)},
	}
	p.wgr.Add(1)
	return p, nil
}

// ListenAndServe starts listening for connections, recording them and proxying
// to the target URL.
func (p *Proxy) ListenAndServe() (err error) {
	if atomic.CompareAndSwapUint32(&p.isrun, 0, 1) {
		defer func() {
			// Ignore "use of closed network connection" comming from closed
			// net.Listener when p was explicitely stopped.
			if !atomic.CompareAndSwapUint32(&p.isrun, 1, 0) {
				err = nil
			}
		}()
		p.m.Lock()
		var l net.Listener
		if l, err = net.Listen("tcp", p.addr); err != nil {
			p.m.Unlock()
			return
		}
		if p.rl, err = newRecListener(l, p.targ, p.Record); err != nil {
			p.m.Unlock()
			return
		}
		p.wgr.Done()
		p.m.Unlock()
		err = p.srv.Serve(p.rl)
		return
	}
	return ErrAlreadyRunning
}

// Addr returns the Proxy's network address. It blocks when the p is not running.
func (p *Proxy) Addr() (addr net.Addr) {
	p.wgr.Wait()
	p.m.Lock()
	addr = p.rl.Addr()
	p.m.Unlock()
	return
}

// Stop stops the Proxy from accepting new connections. It waits for on-going
// connections to finish, ensuring all of them were captured in the l.
func (p *Proxy) Stop() (l *Log, err error) {
	err = ErrNotRunning
	if atomic.CompareAndSwapUint32(&p.isrun, 1, 0) {
		p.wgr.Wait()
		p.m.Lock()
		l, err = &p.rl.log, p.rl.Close()
		p.rl = nil
		p.wgr.Add(1)
		p.m.Unlock()
		return
	}
	return
}
