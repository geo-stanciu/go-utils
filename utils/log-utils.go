package utils

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type logItem struct {
	dt  time.Time
	msg string
}

// AuditLog - Audit log helper
type AuditLog struct {
	mux           *sync.RWMutex
	log           *logrus.Logger
	source        string
	sourceVersion string
	dbutl         *DbUtils
	queue         chan logItem
	wg            *sync.WaitGroup
	query         string
}

// SetWaitGroup - SetWaitGroup
func (a *AuditLog) SetWaitGroup(wg *sync.WaitGroup) {
	a.wg = wg
}

// SetLogger - SetLogger
func (a *AuditLog) SetLogger(source string, sourceVersion string, log *logrus.Logger, dbutl *DbUtils) {
	a.mux = new(sync.RWMutex)
	a.log = log
	a.source = source
	a.sourceVersion = sourceVersion
	a.dbutl = dbutl
	a.queue = make(chan logItem, 10*1024)

	pq := a.dbutl.PQuery(`
		INSERT INTO audit_log (
			log_time,
			source,
			source_version,
			log_msg
		)
		VALUES (?, ?, ?, ?)
	`)

	a.query = pq.Query

	go a.processQueue()
	go a.processQueue()
	go a.processQueue()
	go a.processQueue()
	go a.processQueue()
}

// Close - send signal to close operations
func (a *AuditLog) Close() {
	a.mux.Lock()
	defer a.mux.Unlock()
	close(a.queue)
}

func (a *AuditLog) processQueue() {
	for {
		li, ok := <-a.queue
		if !ok {
			break
		}

		pq := a.dbutl.PQueryNoRewrite(
			a.query,
			li.dt,
			a.source,
			a.sourceVersion,
			li.msg)

		_, err := a.dbutl.Exec(pq)
		if err != nil {
			fmt.Println("log error: ", err)
		}

		if a.wg != nil {
			a.wg.Done()
		}

		time.Sleep(2 * time.Millisecond)
	}
}

func (a AuditLog) Write(p []byte) (n int, err error) {
	if a.wg != nil {
		a.wg.Add(1)
	}

	li := logItem{
		dt:  time.Now().UTC(),
		msg: string(p),
	}

	a.queue <- li

	return len(p), nil
}

// Log - Log Helper function
func (a *AuditLog) Log(err error, msgType string, msg string, details ...interface{}) {
	fields := make(map[string]interface{})

	if len(msgType) > 0 {
		fields["msg_type"] = msgType
	}

	isErr := false
	if err != nil {
		isErr = true
	}

	if isErr {
		fields["status"] = "failed"
	} else {
		fields["status"] = "successful"
	}

	if details != nil {
		var key string

		for i, detail := range details {
			if i%2 == 0 {
				key = detail.(string)
			} else {
				if detail != nil {
					if reflect.ValueOf(detail).Kind() == reflect.Ptr {
						fields[key] = reflect.Indirect(reflect.ValueOf(detail)).Interface()
					} else {
						fields[key] = detail
					}
				}
			}
		}
	}

	hasKeys := false

	if len(fields) > 0 {
		hasKeys = true
	}

	if isErr {
		if hasKeys {
			if err != nil {
				a.log.WithError(err).WithFields(fields).Error(msg)
			} else {
				a.log.WithFields(fields).Error(msg)
			}
		} else {
			if err != nil {
				a.log.WithError(err).Error(msg)
			} else {
				a.log.Error(msg)
			}
		}
	} else {
		if hasKeys {
			if err != nil {
				a.log.WithError(err).WithFields(fields).Info(msg)
			} else {
				a.log.WithFields(fields).Info(msg)
			}
		} else {
			if err != nil {
				a.log.WithError(err).Info(msg)
			} else {
				a.log.Info(msg)
			}
		}
	}
}

func (a *AuditLog) Trace(s string) (string, time.Time) {
    a.Log(nil, "trace", "start", "event", s)
    return s, time.Now()
}

func (a *AuditLog) Un(s string, startTime time.Time) {
    endTime := time.Now()
    a.Log(nil, "trace", "end", "event", s, "elapsed_ms", endTime.Sub(startTime)/1E6)
}
