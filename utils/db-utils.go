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

type DbUtils struct {
    dbType string
}

func (u *DbUtils) SetDbType(dbType string) {
    if len(dbType) == 0 || (dbType != "postgres" && dbType != "oci8" && dbType != "mysql") {
        panic("DbType must be one of: postgres, oci8 or mysql")
    }

    u.dbType = dbType
}

//
// PrepareQuery prepares query for run by changing params written as ? to $1, $2, etc
// for postgres and :1, :2, etc for oracle
//
func (u *DbUtils) PrepareQuery(query string) string {
    if len(u.dbType) == 0 {
        panic("DbType must be one of: postgres, oci8 or mysql")
    }

	q := query
	dbType := strings.ToLower(u.dbType)

	i := 1
	prefix := ""

	if dbType == "postgres" {
		prefix = "$"
	} else if dbType == "oci8" {
		prefix = ":"
	}

	if len(prefix) > 0 {
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
