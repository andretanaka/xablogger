package xablogger

// Segment is the interface that all metric types must implement.
type Segment interface {
	Type() string
	Failed(err error)
	Fields() map[string]interface{}
	HasFailed() bool
	Done()
}
