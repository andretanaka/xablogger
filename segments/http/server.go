package http

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// ServerSegment is used to generate log entries from server perspective. Use this segment when generating metrics
// for your APIs
type ServerSegment struct {
	start time.Time
	data  map[string]interface{}
	mux   sync.Mutex
}

// NewServerSegment initialized a ServerSegment instance. It will set the current timestamp in the segment
func NewServerSegment(r *http.Request) *ServerSegment {
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

	return &ServerSegment{
		start: time.Now(),
		data:  data,
	}
}

// Type returns the segment type
func (s *ServerSegment) Type() string {
	return "http - server"
}

// Failed marks that an error has ocurred on this segment. It will also set an 'status_code' key with
// internal server error (500) status code
func (s *ServerSegment) Failed(err error) {
	s.mux.Lock()
	s.data["error"] = err.Error()
	s.data["status_code"] = http.StatusInternalServerError
	s.mux.Unlock()
}

// Fields return the data fields
func (s *ServerSegment) Fields() map[string]interface{} {
	return s.data
}

// HasFailed returns if the current segment has suffered an error
func (s *ServerSegment) HasFailed() bool {
	return s.data["error"] != nil
}

// JSONResponse sets data for a JSON response. If an error occurs marshalling the body, the 'response_body' key
// will not be set
func (s *ServerSegment) JSONResponse(statusCode int, body interface{}, headers http.Header) {
	s.mux.Lock()

	s.data["status_code"] = statusCode
	s.data["response.headers"] = headers

	if body != nil {
		if responseBytes, err := json.Marshal(body); err == nil {
			s.data["response.body"] = string(responseBytes)
		}
	}
	s.mux.Unlock()
}

// Done stops measuring elapsed time. If the data map does not contains a 'status_code' key, it will set
// the OK (200) status code
func (s *ServerSegment) Done() {
	s.mux.Lock()
	s.data["elapsed_ms"] = int(time.Since(s.start) / time.Millisecond)

	if _, exists := s.data["status_code"]; !exists {
		s.data["status_code"] = http.StatusOK
	}

	s.mux.Unlock()
	return
}
