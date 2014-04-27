package fakerpc

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
)

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
		if rc.rec != nil {
			rc.rec(&rc.t[len(rc.t)-1])
		}
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
	// TODO(rjeczalik): tail might be still empty
	if len(rc.t) > 0 && rc.t[len(rc.t)-1].Src == nil {
		rc.t = rc.t[:len(rc.t)-1]
	}
	rc.commit(rc.t)
	rc.wg.Done()
	return
}

var addrcache = make(map[string]*net.TCPAddr)

func tcpaddr(addr net.Addr) (*net.TCPAddr, error) {
	tcpa, ok := addr.(*net.TCPAddr)
	if ok {
		return tcpa, nil
	}
	tcpa, ok = addrcache[addr.String()]
	if ok {
		return tcpa, nil
	}
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil, err
	}
	tcpa = &net.TCPAddr{}
	if tcpa.Port, err = strconv.Atoi(port); err != nil {
		return nil, err
	}
	if tcpa.IP = net.ParseIP(host); tcpa.IP != nil {
		addrcache[addr.String()] = tcpa
		return tcpa, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	tcpa.IP = ips[0]
	addrcache[addr.String()] = tcpa
	return tcpa, nil
}

type hpwrap string

func (w hpwrap) Network() string { return string(w) }
func (w hpwrap) String() string  { return string(w) }

func urltotcpaddr(u *url.URL) (*net.TCPAddr, error) {
	hp := u.Host
	if _, _, err := net.SplitHostPort(hp); err != nil {
		hp = hp + ":80"
	}
	return tcpaddr(hpwrap(hp))
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
	l = &recListener{
		log: Log{T: make([]Transmission, 0)},
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

func (rl *recListener) Close() (err error) {
	err = rl.lis.Close()
	rl.wg.Wait()
	return
}

func (rl *recListener) Addr() net.Addr {
	return rl.lis.Addr()
}

// Proxy TODO(rjeczalik): document
type Proxy struct {
	Record func(*Transmission)
	m      sync.Mutex
	rl     *recListener
	targ   *url.URL
	addr   string
	srv    *http.Server
	isrun  uint32
}

// NewProxy TODO(rjeczalik): document
func NewProxy(addr, target string) (*Proxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		targ: u,
		addr: addr,
		srv:  &http.Server{Handler: httputil.NewSingleHostReverseProxy(u)},
	}
	return p, nil
}

// ListenAndServe TODO(rjeczalik): document
func (p *Proxy) ListenAndServe() (err error) {
	if atomic.CompareAndSwapUint32(&p.isrun, 0, 1) {
		p.m.Lock()
		defer func() {
			// Ignore "use of closed network connection" comming from closed
			// net.Listener when p was explicitely stopped.
			if !atomic.CompareAndSwapUint32(&p.isrun, 1, 0) {
				err = nil
			}
		}()
		var l net.Listener
		if l, err = net.Listen("tcp", p.addr); err != nil {
			p.m.Unlock()
			return
		}
		if p.rl, err = newRecListener(l, p.targ, p.Record); err != nil {
			p.m.Unlock()
			return
		}
		p.m.Unlock()
		err = p.srv.Serve(p.rl)
		return
	}
	return ErrAlreadyRunning
}

// Running TODO(rjeczalik): document
func (p *Proxy) Running() bool {
	return atomic.LoadUint32(&p.isrun) == 1
}

// Addr TODO(rjeczalik): document
func (p *Proxy) Addr() (addr net.Addr) {
	if p.Running() {
		p.m.Lock()
		addr = p.rl.Addr()
		p.m.Unlock()
	}
	return
}

// Stop TODO(rjeczalik): document
func (p *Proxy) Stop() (l *Log, err error) {
	err = ErrNotRunning
	if atomic.CompareAndSwapUint32(&p.isrun, 1, 0) {
		p.m.Lock()
		l, err = &p.rl.log, p.rl.Close()
		p.rl = nil
		p.m.Unlock()
		return
	}
	return
}
