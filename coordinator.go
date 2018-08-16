package xablogger

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

var coordinatorInstance coordinator

type coordinator struct {
	mainLogger     *logrus.Logger
	defaultFields  map[string]interface{}
	transactionMap map[string]*transaction
}

type transaction struct {
	id       string
	logger   *logrus.Entry
	segments []Segment
	mux      sync.Mutex
}

// Init must be called before creating transactions. It inits all resources like the main logrus Logger, transactionMap and defaultFields fields.
// The xablogger instance can be decorated with functional options provided at this package, like LogFormat, Hooks or DefaultFields
func Init(opts ...func(*coordinator)) {
	coordinatorInstance = coordinator{}
	coordinatorInstance.mainLogger = logrus.New()
	coordinatorInstance.defaultFields = make(map[string]interface{})
	coordinatorInstance.transactionMap = make(map[string]*transaction)

	for _, opt := range opts {
		opt(&coordinatorInstance)
	}

	// sets the default audit=true field so that you can tell which log entries are for whole transactions or not
	coordinatorInstance.defaultFields["audit"] = true
}

// LogFormat sets the output format of the main logger
func LogFormat(formatter logrus.Formatter) func(*coordinator) {
	return func(x *coordinator) {
		x.mainLogger.Formatter = formatter
	}
}

// Hooks is used to add logrus hooks to the coordinator instance of logrus.Logger
func Hooks(hooks ...logrus.Hook) func(*coordinator) {
	return func(x *coordinator) {
		for _, hook := range hooks {
			x.mainLogger.AddHook(hook)
		}
	}
}

// DefaultFields will add entries on the defaultFields property on the coordinator instance. These fields will be added on
// all entries. Use them to set things like environment, app version etc
func DefaultFields(fields map[string]interface{}) func(*coordinator) {
	return func(x *coordinator) {
		for k, v := range fields {
			x.defaultFields[k] = v
		}
	}
}

// NewTransaction creates a new transaction instance at coordinator transactionMap.
// The function will return an error if the transactionMap already contains an entry with the the provided transactionID
func NewTransaction(transactionID string) error {
	tx, exists := coordinatorInstance.transactionMap[transactionID]
	if exists {
		return fmt.Errorf("TransactionID %s already exists inside transactions map", transactionID)
	}

	tx = &transaction{
		id:       transactionID,
		segments: []Segment{},
		logger:   coordinatorInstance.mainLogger.WithFields(coordinatorInstance.defaultFields),
	}

	coordinatorInstance.transactionMap[transactionID] = tx
	return nil
}

// AppendSegment is used to add an extra segment to a given transaction and generate the separate, non-audit log entry for the segment.
// Please note that the whole audit entry will only be generate by calling the Flush function
func AppendSegment(transactionID string, segment Segment) error {

	segmentEntry := coordinatorInstance.mainLogger.WithFields(map[string]interface{}{
		"segment.type": segment.Type(),
		"segment.data": segment.Fields(),
		"audit":        false,
	}).WithFields(coordinatorInstance.defaultFields)

	if segment.HasFailed() {
		segmentEntry.Error()
	} else {
		segmentEntry.Info()
	}

	tx, exists := coordinatorInstance.transactionMap[transactionID]
	if !exists {
		return fmt.Errorf("Transaction %s not found", transactionID)
	}

	tx.mux.Lock()
	tx.segments = append(tx.segments, segment)
	tx.mux.Unlock()
	return nil
}

// FlushTransaction ends a transaction and generates the audit trail log event.
// The function will return an error if the transactionID cannot be found
func FlushTransaction(transactionID string) error {
	tx, exists := coordinatorInstance.transactionMap[transactionID]
	if !exists {
		return fmt.Errorf("TransactionID %s not found", transactionID)
	}

	if _, failed := tx.logger.Data["error"]; failed {
		tx.logger.Error()
	} else {
		tx.logger.Info()
	}

	delete(coordinatorInstance.transactionMap, transactionID)
	return nil
}
