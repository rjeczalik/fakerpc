package black

import (
	"net/http"
	"time"
)

// Record is an intermediate struct for eavesdropping and storing round trip
// http.Request and http.Response struct. It stores timestamps as nanoseconds.
// End of recording is signalized by OnComplete callback.
type Record struct {
	Req struct {
		// Time when Record started eavesdropping on a request body.
		Timestamp int64
		Body      *ClosingBuffer
		*http.Request
	}
	Res struct {
		// Time when the response body has been closed.
		Timestamp int64
		Body      *ClosingBuffer
		*http.Response
	}
	OnComplete func()
}

// RecordRequest eavesdrops on the req body when its ContantLength is eiher
// not set or non-empty.
func (rec *Record) RecordRequest(req *http.Request) {
	rec.Req.Request, rec.Req.Timestamp = req, time.Now().UnixNano()
	if req.ContentLength > 0 || req.ContentLength == -1 {
		rec.Req.Body = new(ClosingBuffer)
		req.Body = TeeReadCloser(req.Body, rec.Req.Body)
	}
}

// RecordResponse eavesdrops on the res body when its ContentLenth is either
// not set or non-empty. It calls OnComplete callback either immadiately when
// the body is empty or after the body is closed.
func (rec *Record) RecordResponse(res *http.Response) {
	var finalize = func() {
		rec.Res.Timestamp = time.Now().UnixNano()
		rec.OnComplete()
	}
	rec.Res.Response = res
	if res != nil && (res.ContentLength > 0 || res.ContentLength == -1) {
		rec.Res.Body = &ClosingBuffer{OnClose: finalize}
		res.Body = TeeReadCloser(res.Body, rec.Res.Body)
		return
	}
	finalize()
}
