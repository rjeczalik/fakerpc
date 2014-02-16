package black

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func dummyRequest(body []byte, contentLen int) (req *http.Request) {
	buf := new(ClosingBuffer)
	req = &http.Request{
		ContentLength: int64(contentLen),
		Body:          buf,
	}
	if n, err := buf.Write(body); err != nil || n != len(body) {
		panic(fmt.Errorf("buf.Write(%q) = %q, %d", body, err, n))
	}
	return
}

func dummyResponse(body []byte, contentLen int) (res *http.Response) {
	buf := new(ClosingBuffer)
	res = &http.Response{
		ContentLength: int64(contentLen),
		Body:          buf,
	}
	if n, err := buf.Write(body); err != nil || n != len(body) {
		panic(fmt.Errorf("buf.Write(%q) = %q, %d", body, err, n))
	}
	return
}

func mockRoundTrip(done chan<- error, rec *Record, req *http.Request, res *http.Response) {
	rec.RecordRequest(req)
	if _, err := io.Copy(ioutil.Discard, req.Body); err != nil {
		done <- err
		return
	}
	rec.RecordResponse(res)
	if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		done <- err
		return
	}
	res.Body.Close()
}

func setupMock(reqBody, resBody []byte, reqLen, resLen int) (*Record, <-chan error) {
	done := make(chan error)
	rec := &Record{OnComplete: func() { done <- nil }}
	req, res := dummyRequest(reqBody, len(reqBody)), dummyResponse(resBody, -1)
	go mockRoundTrip(done, rec, req, res)
	return rec, done
}

func testRecord(t *testing.T, reqBody, resBody []byte, reqLen, resLen int) {
	rec, done := setupMock(reqBody, resBody, reqLen, resLen)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected err to be nil, was %q instead", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("mockRoundTrip has timed out")
	}
	if len(reqBody) > 0 {
		if rec.Req.Body == nil {
			t.Fatalf("expected rec.Req.Body to be non-nil")
		}
	} else {
		if rec.Req.Body != nil {
			t.Fatalf("expected rec.Req.Body to be nil")
		}
	}
	if len(resBody) > 0 {
		if rec.Res.Body == nil {
			t.Fatalf("expected rec.Res.Body to be non-nil")
		}
	} else {
		if rec.Res.Body != nil {
			t.Fatalf("expected rec.Res.Body to be nil")
		}
	}
	if !bytes.Equal(rec.Req.Body.Bytes(), reqBody) {
		t.Errorf("expected rec.Req.Body to be %q, was %q instead", reqBody, rec.Req.Body.Bytes())
	}
	if !bytes.Equal(rec.Res.Body.Bytes(), resBody) {
		t.Errorf("expected rec.Res.Body to be %q, was %q instead", resBody, rec.Res.Body.Bytes())
	}
}

func TestRecord(t *testing.T) {
	table := []struct {
		reqBody, resBody string
		reqLen, resLen   int
	}{
		{"this is request", "no, this is response", 15, -1},
		{"this is another request", "", 24, -1},
		{"", "no, this is also response", 0, -1},
		{"", "", 0, 0},
	}
	for _, row := range table {
		testRecord(t, []byte(row.reqBody), []byte(row.resBody), row.reqLen, row.resLen)
	}
}
