package utils

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type AuditLog struct {
	log     *logrus.Logger
	dbUtils *DbUtils
}

func (a *AuditLog) SetLoggerAndDatabase(log *logrus.Logger, dbUtils *DbUtils) {
	a.log = log
	a.dbUtils = dbUtils
}

func (a AuditLog) Write(p []byte) (n int, err error) {
	query := a.dbUtils.PQuery(`
		INSERT INTO audit_log (
			audit_msg
		)
		VALUES (?)
	`)

	logMsg := string(p)

	_, err = a.dbUtils.db.Exec(query, logMsg)

	if err != nil {
		return 0, fmt.Errorf("log error: %v", err)
	}

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
