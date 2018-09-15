package utils

import (
	"database/sql"
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

	isOracle := u.dbType == Oci8 || u.dbType == Oracle || u.dbType == Oracle11g
	isSqlite := u.dbType == Sqlite3

	if s.columnNames == nil || len(s.columnNames) == 0 {
		cols, err := rows.Columns()
		if err != nil {
			return err
		}

		s.columnNames = cols

		if isOracle {
			for i, colName := range s.columnNames {
				if colName[0:1] != "\"" {
					s.columnNames[i] = strings.ToLower(colName)
				}
			}
		}
	}

	nrCols := len(s.columnNames)
	pointers := make([]interface{}, nrCols)
	altpointers := make([]interface{}, nrCols)
	putback := make([]int, 0)
	fieldTypes := make([]reflect.Type, nrCols)

	structVal := reflect.ValueOf(dest).Elem()
	nFields := structVal.NumField()

	rnum := 0

	dt := time.Now()
	dtnull := NullTime{}
	dtType := reflect.TypeOf(dt)
	dtnullType := reflect.TypeOf(dtnull)

	for i, colName := range s.columnNames {
		if isOracle && colName == "rnumignore" {
			pointers[i] = &rnum
			fieldTypes[i] = reflect.ValueOf(rnum).Type()
			continue
		}

		for j := 0; j < nFields; j++ {
			typeField := structVal.Type().Field(j)
			tag := typeField.Tag

			if tag.Get("sql") == colName {
				pointers[i] = structVal.Field(j).Addr().Interface()
				fieldTypes[i] = typeField.Type

				if isSqlite && (fieldTypes[i] == dtType || fieldTypes[i] == dtnullType) {
					altpointers[i] = pointers[i]
					putback = append(putback, i)
					pointers[i] = new(sql.NullString)
				}

				break
			}
		}
	}

	err := rows.Scan(pointers...)
	if err != nil {
		return err
	}

	if isSqlite {
		np := len(putback)
		for k := 0; k < np; k++ {
			i := putback[k]

			if val, ok := pointers[i].(*sql.NullString); ok && val != nil && (*val).Valid {
				s := (*val).String
				s = strings.Replace(s, "T", " ", 1)
				s = strings.Replace(s, "Z", "", 1)
				l := len(s)

				if l == 0 {
					continue
				}

				if fieldTypes[i] == dtnullType {
					dtval := altpointers[i].(*NullTime)
					(*dtval).SetValue(parseSDate(s, l))
				} else {
					dtval := altpointers[i].(*time.Time)
					*dtval = parseSDate(s, l)
				}
			}
		}
	} else if isOracle {
		// in oci, the timestamp is comming up as local time zone
		// even if you ask for the UTC

		for i := 0; i < nrCols; i++ {
			if fieldTypes[i] == dtType {
				dtval := pointers[i].(*time.Time)
				strdt := Date2string(*dtval, ISODateTimestamp)
				*dtval = String2dateNoErr(strdt, UTCDateTimestamp)
			} else if fieldTypes[i] == dtnullType {
				dtval := pointers[i].(*NullTime)
				if dtval.Valid {
					strdt := Date2string((*dtval).Time, ISODateTimestamp)
					(*dtval).Time = String2dateNoErr(strdt, UTCDateTimestamp)
				}
			}
		}
	}

	return nil
}

func parseSDate(s string, l int) time.Time {
	var dt time.Time

	switch {
	case l == 8:
		dt = String2dateNoErr(s, ISOTime)
	case l == 10:
		dt = String2dateNoErr(s, UTCDate)
	case l == 12:
		dt = String2dateNoErr(s, ISOTimeMS)
	case l == 19:
		dt = String2dateNoErr(s, UTCDateTime)
	case l >= 23:
		s = s[0:23]
		dt = String2dateNoErr(s, UTCDateTimestamp)
	}

	return dt
}
