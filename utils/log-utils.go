package utils

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type logItem struct {
	dt  time.Time
	msg string
}

type AuditLog struct {
	log       *logrus.Logger
	logSource string
	dbUtils   *DbUtils
	queue     chan logItem
	wg        *sync.WaitGroup
}

func (a *AuditLog) SetWaitGroup(wg *sync.WaitGroup) {
	a.wg = wg
}

func (a *AuditLog) SetLogger(logSource string, log *logrus.Logger, dbUtils *DbUtils) {
	a.log = log
	a.logSource = logSource
	a.dbUtils = dbUtils
	a.queue = make(chan logItem, 128)
}

func (a *AuditLog) processQueue() {
	if a.wg != nil {
		defer a.wg.Done()
	}

	li := <-a.queue

	query := a.dbUtils.PQuery(`
		INSERT INTO audit_log (
			log_time, log_source, audit_msg
		)
		VALUES (?, ?, ?)
	`)

	_, err := a.dbUtils.db.Exec(query, li.dt, a.logSource, li.msg)

	if err != nil {
		fmt.Println("log error: ", err)
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

	go a.processQueue()

	return len(p), nil
}

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
				fields[key] = detail
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
