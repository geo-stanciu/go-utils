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
	dateformats []string
}

// Clear - clears the columns array.
// Used to be able to reuse the scan helper for another SQL
func (s *SQLScan) Clear() {
	s.Lock()
	defer s.Unlock()

	s.columnNames = nil
	s.dateformats = nil
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
				sdt := (*val).String

				sdt = strings.Replace(sdt, "T", " ", 1)
				sdt = strings.Replace(sdt, "Z", "", 1)
				l := len(sdt)

				if l == 0 {
					continue
				}

				val, err := s.parseSDate(sdt)
				if err != nil {
					return err
				}

				if fieldTypes[i] == dtnullType {
					dtval := altpointers[i].(*NullTime)
					(*dtval).SetValue(val)
				} else {
					dtval := altpointers[i].(*time.Time)
					*dtval = val
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

func padRight(str string, item string, count int) string {
	return str + strings.Repeat(item, count-len(str))
}

func (s *SQLScan) parseSDate(sdt string) (time.Time, error) {
	var dt time.Time
	var err error
	var err1 error
	found := false

	if s.dateformats == nil || len(s.dateformats) == 0 {
		s.dateformats = []string{
			"2006-01-02 15:04:05.000000000Z07:00",
			"2006-01-02 15:04:05.000000000",
			"2006-01-02 15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"2006-01-02 15:04",
			"2006-01-02",
			"15:04:05.000000000Z07:00",
			"15:04:05.000000000",
			"15:04:05Z07:00",
			"15:04:05",
			"15:04",
		}
	}

	// transform in <date time.nano seconds>  format
	sdate := sdt
	idx := strings.Index(sdt, ".")

	if idx > 0 {
		idx2 := strings.Index(sdt[idx:], "+")

		if idx2 > 0 {
			sdate = fmt.Sprintf("%v%v%v", sdt[0:idx+1], padRight(sdt[idx+1:idx+idx2], "0", 9), sdt[idx+idx2:])
		} else {
			idx2 = strings.Index(sdt[idx:], "-")

			if idx2 > 0 {
				sdate = fmt.Sprintf("%v%v%v", sdt[0:idx+1], padRight(sdt[idx+1:idx+idx2], "0", 9), sdt[idx+idx2:])
			} else {
				sdate = fmt.Sprintf("%v%v", sdt[0:idx+1], padRight(sdt[idx+1:], "0", 9))
			}
		}
	}
	
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		return time.Now(), err
	}

	for _, format := range s.dateformats {
		dt, err1 = time.ParseInLocation(format, sdate, loc)

		if err1 == nil {
			found = true
			break
		}

		if err == nil {
			err = err1
		}

		continue
	}

	if !found {
		if err == nil {
			err = fmt.Errorf("Unknown date format: \"%s\"", sdt)
		}

		return dt, err
	}

	return dt, nil
}
