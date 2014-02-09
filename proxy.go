package black

import (
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// RecordingProxy TODO doc
type RecordingProxy struct {
	proxy *httputil.ReverseProxy
	d     *dumper
}

// NewRecordingProxy TODO doc
func NewRecordingProxy(w io.WriteCloser) *RecordingProxy {
	return &RecordingProxy{
		d: newDumper(w),
	}
}

// TODO
func (rp *RecordingProxy) Close() (err error) {
	return
}

// TODO
func (rp *RecordingProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rp.proxy.ServeHTTP(rw, req)
}

// ListenAndServe TODO doc
func (rp *RecordingProxy) ListenAndServe(addr, target string) error {
	if u, err := url.Parse(target); err == nil {
		rp.proxy = httputil.NewSingleHostReverseProxy(u)
		rp.proxy.Transport = newRecordingTransport(rp.d)
	} else {
		return err
	}
	return http.ListenAndServe(addr, rp.proxy)
}
