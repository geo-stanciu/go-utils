package utils

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type logItem struct {
	exitSignal bool
	dt         time.Time
	msg        string
}

// AuditLog - Audit log helper
type AuditLog struct {
	log        *logrus.Logger
	logSource  string
	dbUtils    *DbUtils
	queue      chan logItem
	wg         *sync.WaitGroup
	exitSignal bool
}

// SetWaitGroup - SetWaitGroup
func (a *AuditLog) SetWaitGroup(wg *sync.WaitGroup) {
	a.wg = wg
}

// SetLogger - SetLogger
func (a *AuditLog) SetLogger(logSource string, log *logrus.Logger, dbUtils *DbUtils) {
	a.log = log
	a.logSource = logSource
	a.dbUtils = dbUtils
	a.queue = make(chan logItem, 1024)

	go a.processQueue()
	go a.processQueue()
	go a.processQueue()
	go a.processQueue()
}

// Close - send signal to close operations
func (a *AuditLog) Close() {
	if a.exitSignal {
		return
	}

	a.exitSignal = true

	li := logItem{
		exitSignal: true,
		dt:         time.Now().UTC(),
		msg:        "exit",
	}

	a.queue <- li
	a.queue <- li
	a.queue <- li
	a.queue <- li
}

func (a *AuditLog) processQueue() {
	for {
		li := <-a.queue

		if li.exitSignal {
			break
		}

		pq := a.dbUtils.PQuery(`
			INSERT INTO audit_log (
				log_time, log_source, audit_msg
			)
			VALUES (?, ?, ?)
		`, li.dt,
			a.logSource,
			li.msg)

		_, err := a.dbUtils.Exec(pq)
		if err != nil {
			fmt.Println("log error: ", err)
		}

		if a.wg != nil {
			a.wg.Done()
		}
	}
}

func (a AuditLog) Write(p []byte) (n int, err error) {
	if a.exitSignal {
		return 0, fmt.Errorf("exit signal already received")
	}

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
func (a AuditLog) Log(err error, msgType string, msg string, details ...interface{}) {
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
