package black

import (
	"encoding/gob"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type proxy struct {
	ch   chan error
	l    net.Listener
	h    Header
	s    Session
	srv  *http.Server
	enc  *gob.Encoder
	file string
}

// Run TODO
func (p *proxy) Run() error {
	log.Printf("proxy server listening on %s . . .", p.l.Addr())
	return p.srv.Serve(p.l)
}

// Close TODO
func (p *proxy) Close() error {
	p.l.Close()
	log.Printf("saving session as %s . . .", p.file)
	// DEBUG
	log.Printf("[DEBUG] len(p.s) = %d", len(p.s))
	for i, c := range p.s {
		log.Printf("[DEBUG] len(p.s[%d]) = %d", i, len(c))
		for i, m := range c {
			log.Printf("[DEBUG] [%d] p.s[%d].Raw.Len() = %d", m.Type, i, m.Raw.Len())
		}
	}
	// /DEBUG
	return p.enc.Encode(&SessionFile{&p.h, &p.s})
}

// NewProxy TODO
func NewProxy(proxyurl, addr string, file *os.File) (RunCloser, error) {
	if !strings.HasPrefix(proxyurl, "http://") && !strings.HasPrefix(proxyurl, "https://") {
		proxyurl = "http://" + proxyurl
	}
	u, err := url.Parse(proxyurl)
	if err != nil {
		return nil, err
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newProxy(NewHeader("http", proxyurl), u, l, file), nil
}

func newProxy(h Header, u *url.URL, l net.Listener, file *os.File) RunCloser {
	p := &proxy{
		ch:   make(chan error, 1),
		h:    h,
		s:    NewSession(),
		srv:  &http.Server{Handler: httputil.NewSingleHostReverseProxy(u)},
		enc:  gob.NewEncoder(file),
		file: file.Name(),
	}
	p.l = p.s.Wiretap(l)
	return p
}
