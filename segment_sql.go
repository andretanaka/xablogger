package xablogger

import (
	"database/sql"
	"sync"
	"time"
)

// SQLSegment is used to generate log entries from SQL transactions.
type SQLSegment struct {
	start time.Time
	data  map[string]interface{}
	mux   sync.Mutex
}

// NewSQLSegment initializes a SQLSegment instance. It will set the current timestamp in the segment start data and latency
// will be computed from the function return until Done function is called.
func NewSQLSegment(driver string, statement string, params map[string]interface{}) *SQLSegment {
	return &SQLSegment{
		start: time.Now(),
		data: map[string]interface{}{
			"statement": statement,
			"params":    params,
			"driver":    driver,
		},
	}
}

// Type returns the segment type
func (s *SQLSegment) Type() string {
	return "sql"
}

// Failed marks that an error has ocurred on this segment.
func (s *SQLSegment) Failed(err error) {
	s.mux.Lock()
	s.data["error"] = err.Error()
	s.mux.Unlock()
}

// Fields return the data fields
func (s *SQLSegment) Fields() map[string]interface{} {
	return s.data
}

// HasFailed returns if the current segment has suffered an error
func (s *SQLSegment) HasFailed() bool {
	return s.data["error"] != nil
}

// ExecResponse fills the keys for an Exec response, setting the number of affected rows.
// If an error occurs, it will skip the affected rows field
func (s *SQLSegment) ExecResponse(res sql.Result) {
	s.mux.Lock()
	if rowsAffected, err := res.RowsAffected(); err == nil {
		s.data["rows_affected"] = rowsAffected
	}
	s.mux.Unlock()
}

// QueryResponse fills the keys for a Query function response. As I did not found any way to copying the rows object,
// this function only logs the columns of the result set.
// Note that this function does not closes the rows object, you need to do it yourself.
func (s *SQLSegment) QueryResponse(rows *sql.Rows) {
	s.mux.Lock()
	if columns, err := rows.Columns(); err == nil {
		s.data["columns"] = columns
	}
	s.mux.Unlock()
}

// Done stops measuring elapsed time
func (s *SQLSegment) Done() {
	s.mux.Lock()
	s.data["elapsed_ms"] = int(time.Since(s.start) / time.Millisecond)
	s.mux.Unlock()
	return
}
