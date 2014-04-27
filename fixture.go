package fakerpc

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var re = regexp.MustCompile(`.*\.(Test.*)`)

func Fixture(t *testing.T) (addr string, teardown func()) {
	pc := make([]uintptr, 10)
	runtime.Callers(1, pc)
	var logfile string
	for _, pc := range pc {
		if f := runtime.FuncForPC(pc); f != nil {
			if m := re.FindStringSubmatch(f.Name()); len(m) == 2 {
				file, _ := f.FileLine(pc)
				logfile = filepath.Join(filepath.Dir(file), "testdata", strings.ToLower(m[1])+".gob")
			}
		}
	}
	if logfile == "" {
		t.Fatal("fakerpc: unable to guess the path to a log file for this test")
	}
	var (
		rpcaddr string
		record  bool
	)
	if rpcaddr = os.Getenv("FAKERPC_RECORD"); rpcaddr != "" {
		record = true
	} else {
		rpcaddr = os.Getenv("FAKERPC")
	}
	if rpcaddr != "" {
		p, err := NewProxy("localhost:0", rpcaddr)
		if err != nil {
			t.Fatal("fakerpc: unable to create proxy:", err)
		}
		go func() {
			if err := p.ListenAndServe(); err != nil {
				t.Fatal("fakerpc: proxy error:", err)
			}
		}()
		for !p.Running() {
		}
		addr = "http://" + p.Addr().String()
		teardown = func() {
			l, err := p.Stop()
			if err != nil {
				t.Fatal("fakerpc: proxy teardown error:", err)
			}
			if record {
				if err = os.MkdirAll(filepath.Dir(logfile), 0755); err != nil {
					t.Fatal("fakerpc: error creating testdata dir:", err)
				}
				if err = WriteLog(logfile, l); err != nil {
					t.Fatal("fakerpc: error writing log file:", err)
				}
			}
		}
	} else {
		l, err := ReadLog(logfile)
		if err != nil {
			t.Fatal("fakerpc: error reading log file:", err)
		}
		srv, err := NewServer("localhost:0", l)
		if err != nil {
			t.Fatal("fakerpc: unable to create server:", err)
		}
		go func() {
			if err := srv.ListenAndServe(); err != nil {
				t.Fatal("fakerpc: server error:", err)
			}
		}()
		for !srv.Running() {
		}
		addr, teardown = "http://"+srv.Addr().String(), func() { srv.Stop() }
	}
	return
}
