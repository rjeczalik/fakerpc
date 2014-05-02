package fakerpc

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"runtime"
	"testing"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func httpsrv(t *testing.T) string {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, req.Body); err != nil {
			t.Error(err)
			return
		}
		req.Body.Close()
		if _, err := io.Copy(w, &buf); err != nil {
			t.Error(err)
		}
	})
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		if err := http.Serve(l, nil); err != nil {
			t.Fatal(err)
		}
	}()
	return l.Addr().String()
}

func mul(p []byte, n int) []byte {
	q := make([]byte, 0, n*len(p))
	for ; n > 0; n-- {
		q = append(q, p...)
	}
	return q
}

func TestProxy(t *testing.T) {
	addr := httpsrv(t)
	p, err := NewProxy(":0", "http://"+addr)
	if err != nil {
		t.Fatalf("expected err=nil; got %q", err)
	}
	go func() {
		if err := p.ListenAndServe(); err != nil {
			t.Fatal(err)
		}
	}()
	body := [][][]byte{{
		mul([]byte("1st reqeust"), 256),
		mul([]byte("2nd request"), 256),
		mul([]byte("3rd request"), 256),
	}, {
		mul([]byte("Hello"), 1024),
		mul([]byte("world"), 1024),
	}, {
		mul([]byte("S7Dz5rYzc6bXInLFaKUAFPxqpYDRrBfMsR1ujle61eNCTKuX6K9MLDeDGwWKVB"+
			"yrtZR6EBA3BQndVAVlhOBQrcHnbVzy64PEywFPfhsHJQXf6XfBcrYwh7o3KlUT"+
			"edh5bRon30krmvoOWIhqdnMGhy5wg2Hj84c9frDxC3JxPZZpvIUdgewhSvRXO5"+
			"PgOUx4ZAW8jLnb9mvZbdRfTvbjfjn6jxqgiIMW4xsfJ9xHOFgeDDFStL2iKltv"), 128),
	}, {
		mul([]byte("A"), 10240),
		mul([]byte("C"), 10240),
		mul([]byte("D"), 10240),
		mul([]byte("E"), 10240),
		mul([]byte("F"), 10240),
	}}
	last, all := 0, make([][]byte, 0)
	for _, body := range body {
		var c http.Client
		last, all = len(body)-1, append(all, body...)
		for i, body := range body {
			r, err := http.NewRequest("POST", "http://"+p.Addr().String(), bytes.NewBuffer(body))
			if err != nil {
				t.Errorf("expected err=nil; got %q (i=%d)", err, i)
				continue
			}
			r.Close = (i == last)
			res, err := c.Do(r)
			if err != nil {
				t.Errorf("expected err=nil; got %q (i=%d)", err, i)
				continue
			}
			buf := bytes.NewBuffer(make([]byte, 0, len(body)))
			if _, err = io.Copy(buf, res.Body); err != nil {
				t.Errorf("expected err=nil; got %q (i=%d)", err, i)
				continue
			}
			if buf.Len() != len(body) {
				t.Errorf("expected buf.Len()=%d; got %d", len(body), buf.Len())
				continue
			}
			if !bytes.Equal(buf.Bytes(), body) {
				t.Errorf("expected res.Body=%q; got %q", buf.Bytes(), body)
			}
		}
	}
	log, err := p.Stop()
	if err != nil {
		t.Fatalf("expected err=nil; got %q", err)
	}
	if len(log.T) != len(all)*2 {
		t.Errorf("expected len(log.T)=%d; got %d", len(all)*2, len(log.T))
	}
	for i := range all {
		j := 2 * i
		header, body := SplitHeaderBody(log.T[j].Raw)
		if header == nil {
			t.Errorf("expected header!=nil (log.T[%d].Raw)", j)
			continue
		}
		if body == nil {
			t.Errorf("expected body!=nil (log.T[%d].Raw)", j)
			continue
		}
		if len(body) != len(all[i]) {
			t.Errorf("expected len(body)=%d; got %d (log.T[%d].Raw)",
				len(all[i]), len(body), j)
			continue
		}
		if !bytes.Equal(body, all[i]) {
			t.Errorf("expected body==all[%d] (log.T[%d].Raw)", i, j)
		}
	}
	c, err := NewConnections(log)
	if err != nil {
		t.Fatalf("expected err=nil; got %q", err)
	}
	if len(c) != len(body) {
		t.Fatalf("expected len(c)=%d; got %d", len(body), len(c))
	}
	for i, c := range c {
		if len(c) != len(body[i]) {
			t.Errorf("expected len(c[%d])=%d; got %d", i, len(body[i]), len(c))
			continue
		}
		for j, c := range c {
			if c.Req.Method != "POST" {
				t.Errorf(`expected c[%d][%d].Req.Method="POST"; got %q`, i, j,
					c.Req.Method)
			}
			if len(c.ReqBody) != len(body[i][j]) {
				t.Errorf("expected len(c[%d][%d].ReqBody)=%d; got %d", i, j,
					len(body[i][j]), len(c.ReqBody))
			}
			if len(c.Res) < len(body[i][j]) {
				t.Errorf("expected len(c[%d][%d].Res)>%d; got %d", i, j,
					len(body[i][j]), len(c.Res))
			}
		}
	}
}
