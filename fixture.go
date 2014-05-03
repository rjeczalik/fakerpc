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

// Fixture provides a fake server for mocking HTTP-based RPC services. A single
// fixture has TestXxx-function scope, which makes it sutiable for parallel test
// execution. Typical usage of the Fixture function is to setup the mock at the
// begining of the test, deferring returned "teardown" func; the "addr" return
// value is new network address of the RPC service being tested. Example:
//
//   func TestName(t *testing.T) {
//     addr, teardown := fakerpc.Fixture(t)
//     defer teardown()
//     // ...
//     client, err := rpc.DialHTTP("tcp", addr)
//     // ...
//
// Control
//
// The default behavior of the fake is to reply to requests issued within a test
// with previously-recorded responses. The fake looks up the record-log file
// under the ./testdata/{{.testname}}.gzob which is relative to the *_test.go file
// from which the Fixture was called. The ".testname" is a lower-cased TestXxx-
// function name.
//
// A mock server created by the Fixture can be configured to act as:
//
//  * a reply-server (default behavior)
//  * a proxy service by setting the target URL with FAKERPC environment variable
//  * a recording proxy service - accordingly FAKERPC_RECORD environment variable
//
// In order to create record-log files for the first use of a reply-server, run
// the tests with a FAKERPC_RECORD environment variable pointing to your service's
// end point. Example:
//
//   FAKERPC_RECORD="http://rpc.int.mycompany.com:8079" go test ./...
func Fixture(t *testing.T) (addr string, teardown func()) {
	pc := make([]uintptr, 10)
	runtime.Callers(1, pc)
	var logfile string
	for _, pc := range pc {
		if f := runtime.FuncForPC(pc); f != nil {
			if m := re.FindStringSubmatch(f.Name()); len(m) == 2 {
				file, _ := f.FileLine(pc)
				logfile = filepath.Join(filepath.Dir(file), "testdata", strings.ToLower(m[1])+".gzob")
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
		addr, teardown = "http://"+srv.Addr().String(), func() { srv.Stop() }
	}
	return
}
