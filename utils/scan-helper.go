package utils

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// SQLScanHelper helper class for reading sql to Struct
// Columns must be named the same
// Ex: in sql a column name is col1, in struct must be Col1
type SQLScanHelper struct {
	sync.RWMutex
	columnNames []string
}

// Clear - clears the columns array.
// Used to be able to reuse the scan helper for another SQL
func (s *SQLScanHelper) Clear() {
	s.RLock()
	defer s.RUnlock()

	s.columnNames = nil
}

// Scan - reads sql statement into a struct
func (s *SQLScanHelper) Scan(u *DbUtils, rows *sql.Rows, dest interface{}) error {
	s.RLock()
	defer s.RUnlock()

	if s.columnNames == nil || len(s.columnNames) == 0 {
		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		s.columnNames = cols
	}

	pointers := make([]interface{}, len(s.columnNames))
	structVal := reflect.ValueOf(dest)

	for i, colName := range s.columnNames {
		loweredcolname := strings.ToLower(colName)
		fieldVal := structVal.Elem().FieldByName(strings.Title(loweredcolname))

		if !fieldVal.IsValid() {
			return fmt.Errorf(colName + " field not valid")
		}
		if fieldVal.CanSet() {
			pointers[i] = fieldVal.Addr().Interface()
		}
	}

	err := rows.Scan(pointers...)
	if err != nil {
		return err
	}

	if u.dbType == Oracle {
		// in oci, the timestamp is comming up as local time zone
		// even if you ask for the UTC
		dt := time.Now()

		for _, colName := range s.columnNames {
			loweredcolname := strings.ToLower(colName)
			fieldVal := structVal.Elem().FieldByName(strings.Title(loweredcolname))

			if fieldVal.IsValid() && fieldVal.Type() == reflect.TypeOf(dt) {
				dtval := fieldVal.Addr().Interface().(*time.Time)
				strdt := Date2string(*dtval, ISODateTimestamp)
				*dtval = String2dateNoErr(strdt, UTCDateTimestamp)
			}
		}
	}

	return nil
}
