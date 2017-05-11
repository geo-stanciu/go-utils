package utils

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// NullTime represents a time.Time that may be null. NullTime implements the
// sql.Scanner interface so it can be used as a scan destination, similar to
// sql.NullString.
type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	nt.Time, nt.Valid = value.(time.Time)
	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}

func PrepareQuery(query string) string {
	q := query
	dbType := strings.ToLower(config.DbType)

	i := 1
	prefix := ""

	if dbType == "postgres" {
		prefix = "$"
	} else if dbType == "oci8" {
		prefix = ":"
	}

	if dbType != "mysql" {
		for {
			idx := strings.Index(q, "?")

			if idx < 0 {
				break
			}

			prm := fmt.Sprintf("%s%d", prefix, i)
			i++

			q = strings.Replace(q, "?", prm, 1)
		}
	}

	return q
}
