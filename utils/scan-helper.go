package utils

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// SQLScan helper class for reading sql to Struct
// Columns in struct must be marked with a `sql:"col_name"` tag
// Ex: in sql a column name is col1, in struct the col tag must be `sql:"col1"`
type SQLScan struct {
	sync.RWMutex
	columnNames []string
}

// Clear - clears the columns array.
// Used to be able to reuse the scan helper for another SQL
func (s *SQLScan) Clear() {
	s.Lock()
	defer s.Unlock()

	s.columnNames = nil
}

// Scan - reads sql statement into a struct
func (s *SQLScan) Scan(u *DbUtils, rows *sql.Rows, dest interface{}) error {
	s.Lock()
	defer s.Unlock()

	if s.columnNames == nil || len(s.columnNames) == 0 {
		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		s.columnNames = cols

		if u.dbType == Oci8 || u.dbType == Oracle || u.dbType == Oracle11g {
			for i, colName := range s.columnNames {
				if colName[0:1] != "\"" {
					s.columnNames[i] = strings.ToLower(colName)
				}
			}
		}
	}

	nrCols := len(s.columnNames)
	pointers := make([]interface{}, nrCols)
	fieldTypes := make([]reflect.Type, nrCols)

	structVal := reflect.ValueOf(dest).Elem()
	nFields := structVal.NumField()

	rnum := 0
	rnumPos := -1

	for i, colName := range s.columnNames {
		if u.dbType == Oracle && colName == "rnumignore" {
			rnumPos = i
			pointersAux := make([]interface{}, nrCols+1)

			for k := 0; k < i; k++ {
				pointersAux[k] = pointers[k]
			}

			pointersAux[i] = &rnum

			for k := i; k < nrCols; k++ {
				pointersAux[k+1] = pointers[k]
			}

			pointers = pointersAux

			continue
		}

		for j := 0; j < nFields; j++ {
			typeField := structVal.Type().Field(j)
			tag := typeField.Tag

			if tag.Get("sql") == colName {
				pointers[i] = structVal.Field(j).Addr().Interface()
				fieldTypes[i] = typeField.Type
				break
			}
		}
	}

	fmt.Println("pointers", pointers)

	err := rows.Scan(pointers...)
	if err != nil {
		return err
	}

	if u.dbType == Oci8 || u.dbType == Oracle || u.dbType == Oracle11g {
		// in oci, the timestamp is comming up as local time zone
		// even if you ask for the UTC
		dt := time.Now()
		dtnull := NullTime{}
		dtType := reflect.TypeOf(dt)
		dtnullType := reflect.TypeOf(dtnull)

		for i := 0; i < nrCols; i++ {
			if fieldTypes[i] == dtType {
				if rnumPos < 0 {
					dtval := pointers[i].(*time.Time)
					strdt := Date2string(*dtval, ISODateTimestamp)
					*dtval = String2dateNoErr(strdt, UTCDateTimestamp)
				} else if i >= rnumPos {
					dtval := pointers[i+1].(*time.Time)
					strdt := Date2string(*dtval, ISODateTimestamp)
					*dtval = String2dateNoErr(strdt, UTCDateTimestamp)
				}
			} else if fieldTypes[i] == dtnullType {
				if rnumPos < 0 {
					dtval := pointers[i].(*NullTime)
					if dtval.Valid {
						strdt := Date2string((*dtval).Time, ISODateTimestamp)
						(*dtval).Time = String2dateNoErr(strdt, UTCDateTimestamp)
					}
				} else if i >= rnumPos {
					dtval := pointers[i+1].(*NullTime)
					if dtval.Valid {
						strdt := Date2string((*dtval).Time, ISODateTimestamp)
						(*dtval).Time = String2dateNoErr(strdt, UTCDateTimestamp)
					}
				}
			}
		}
	}

	return nil
}
