package utils

import (
	"database/sql"
	"reflect"
	"strings"
	"sync"
	"time"
)

// SQLScanHelper helper class for reading sql to Struct
// Columns in struct must be marked with a `sql:"col_name"` tag
// Ex: in sql a column name is col1, in struct the col tag must be `sql:"col1"`
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

	nrCols := len(s.columnNames)
	pointers := make([]interface{}, nrCols)
	fieldTypes := make([]reflect.Type, nrCols)

	structVal := reflect.ValueOf(dest).Elem()
	nFields := structVal.NumField()

	for i, colName := range s.columnNames {
		loweredColName := strings.ToLower(colName)
		for j := 0; j < nFields; j++ {
			typeField := structVal.Type().Field(j)
			tag := typeField.Tag

			if tag.Get("sql") == loweredColName {
				pointers[i] = structVal.Field(j).Addr().Interface()
				fieldTypes[i] = typeField.Type
				break
			}
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
		dtType := reflect.TypeOf(dt)

		for i := 0; i < nrCols; i++ {
			if fieldTypes[i] == dtType {
				dtval := pointers[i].(*time.Time)
				strdt := Date2string(*dtval, ISODateTimestamp)
				*dtval = String2dateNoErr(strdt, UTCDateTimestamp)
			}
		}
	}

	return nil
}
