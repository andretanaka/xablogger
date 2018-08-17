package xablogger

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// HTTPSegment is used to generate log entries from HTTP transactions. Use this segment for generating log metrics
// for incoming requests for your REST API or when invoking external APIs via HTTP clients
type HTTPSegment struct {
	start time.Time
	data  map[string]interface{}
	mux   sync.Mutex
}

// NewServerSegment initialized a ServerSegment instance. It will set the current timestamp in the segment start data and latency
// will be computed from the function return until Done function is called.
// It will also use the body contents and set an identifical copy on the request object.
func NewServerSegment(r *http.Request) *HTTPSegment {
	// init data map with all default fields
	data := map[string]interface{}{
		"method":               r.Method,
		"path":                 r.URL.Path,
		"request.query_params": r.URL.Query,
		"request.headers":      r.Header,
	}

	// if the request has body, we duplicate the buffer so that we can log the body contents and
	// keep the request unmodified
	if r.Body != http.NoBody {
		buf, _ := ioutil.ReadAll(r.Body)
		// sets the copied buffer on the request
		r.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		data["request.body"] = string(buf)
	}

	return &HTTPSegment{
		start: time.Now(),
		data:  data,
	}
}

// Type returns the segment type
func (s *HTTPSegment) Type() string {
	return "http"
}

// Failed marks that an error has ocurred on this segment. It will also set an 'status_code' key with
// internal server error (500) status code
func (s *HTTPSegment) Failed(err error) {
	s.mux.Lock()
	s.data["error"] = err.Error()
	s.data["status_code"] = http.StatusInternalServerError
	s.mux.Unlock()
}

// Fields return the data fields
func (s *HTTPSegment) Fields() map[string]interface{} {
	return s.data
}

// HasFailed returns if the current segment has suffered an error
func (s *HTTPSegment) HasFailed() bool {
	return s.data["error"] != nil
}

// Response fills the keys for response data.
// It will also use the body contents and set an identifical copy on the response object.
func (s *HTTPSegment) Response(res *http.Response) {
	s.mux.Lock()
	s.data["status_code"] = res.StatusCode
	s.data["response.headers"] = res.Header

	// if the response has body, we duplicate the buffer so that we can log the body contents and
	// keep the response unmodified
	if res.Body != http.NoBody {
		buf, _ := ioutil.ReadAll(res.Body)
		// sets the copied buffer on the request
		res.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
		s.data["response.body"] = string(buf)
	}

	s.mux.Unlock()
}

// Done stops measuring elapsed time. If the data map does not contains a 'status_code' key, it will set
// the OK (200) status code
func (s *HTTPSegment) Done() {
	s.mux.Lock()
	s.data["elapsed_ms"] = int(time.Since(s.start) / time.Millisecond)

	if _, exists := s.data["status_code"]; !exists {
		s.data["status_code"] = http.StatusOK
	}

	s.mux.Unlock()
	return
}
