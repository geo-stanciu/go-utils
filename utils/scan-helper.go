package utils

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
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
func (s *SQLScanHelper) Scan(rows *sql.Rows, dest interface{}) error {
	s.RLock()
	defer s.RUnlock()

	if len(s.columnNames) == 0 {
		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		s.columnNames = cols
	}

	pointers := make([]interface{}, len(s.columnNames))
	structVal := reflect.ValueOf(dest)

	for i, colName := range s.columnNames {
		fieldVal := structVal.Elem().FieldByName(strings.Title(colName))
		if !fieldVal.IsValid() {
			return fmt.Errorf(colName + "field not valid")
		}
		if fieldVal.CanSet() {
			pointers[i] = fieldVal.Addr().Interface()
		}
	}

	err := rows.Scan(pointers...)
	if err != nil {
		return err
	}

	return nil
}
