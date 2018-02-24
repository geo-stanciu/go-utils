package utils

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const (
	// Postgres - defines PostgreSQL sql driver name
	Postgres string = "postgres"
	// Oracle - defines Oracle sql driver name - for Oracle 12c+
	Oracle string = "oracle"
	// Oci8 - defines Oracle sql driver name - for Oracle 12c+
	Oci8 string = "oci8"
	// Oracle11g - defines Oracle sql driver name - for Oracle11g
	Oracle11g string = "oracle11g"
	// MySQL - defiens MySQL driver name
	MySQL string = "mysql"
	// SQLServer - defines Microsoft SQL Server driver name
	SQLServer string = "mssql"
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

// DbUtils can be used to prepare queries by changing the sql param notations
// as defined by each supported database
type DbUtils struct {
	db     *sql.DB
	dbType string
	prefix string
}

func (u *DbUtils) setDbType(dbType string) {
	dbtypes := []string{
		Postgres,
		Oci8,
		Oracle,
		Oracle11g,
		MySQL,
		SQLServer,
	}

	if len(dbType) == 0 || !stringInSlice(dbType, dbtypes) {
		panic("DbType must be one of: " + strings.Join(dbtypes, ", "))
	}

	u.dbType = strings.ToLower(dbType)

	switch u.dbType {
	case Postgres:
		u.prefix = "$"
	case Oci8:
		u.prefix = ":"
	case Oracle:
		u.prefix = ":"
	case Oracle11g:
		u.prefix = ":"
	default:
		u.prefix = ""
	}
}

// PQuery prepares query for running.
// Query parameter placeholders will be written as ? in all suported databses.
//   Ex: select col1 from table1 where col2 = ?
// Some alterations to the query will be made:
//   - get dates as UTC
//   - in Postgresql
//       - changes params written as ? to $1, $2, etc
//   - in MySQL
//       - replaces quote identifiers with backticks
//   - in SQL Server
//       - replaces "LIMIT ? OFFSET ?" with "OFFSET ? ROWS FETCH NEXT ? ROWS ONLY"
//       - switches parameters set for OFFSET and LIMIT to reflect the changed query
//       - Limitations:
//           - LIMIT ? OFFSET ? must be the last 2 parameters in the query
//   - in Oracle
//       - changes params written as ? to :1, :2, etc
func (u *DbUtils) PQuery(query string, args ...interface{}) *PreparedQuery {
	pq := PreparedQuery{
		DbType:      u.dbType,
		ParamPrefix: u.prefix,
		Query:       query,
		Args:        args,
	}
	pq.Prepare()

	return &pq
}

// Connect2Database - connect to a database
func (u *DbUtils) Connect2Database(db **sql.DB, dbType, dbURL string) error {
	var err error
	u.setDbType(dbType)

	if dbType == Oracle11g || dbType == Oracle {
		*db, err = sql.Open(Oci8, dbURL)
	} else {
		*db, err = sql.Open(dbType, dbURL)
	}

	if err != nil {
		return errors.New("Can't connect to the database, go error " + fmt.Sprintf("%s", err))
	}

	err = (*db).Ping()
	if err != nil {
		return errors.New("Can't ping the database, go error " + fmt.Sprintf("%s", err))
	}

	u.db = *db

	return nil
}

// Exec - exec query without result
func (u *DbUtils) Exec(pq *PreparedQuery) (sql.Result, error) {
	res, err := u.db.Exec(pq.Query, pq.Args...)
	if err != nil {
		return res, err
	}

	return res, nil
}

// ExecTx - exec query without result
func (u *DbUtils) ExecTx(tx *sql.Tx, pq *PreparedQuery) (sql.Result, error) {
	res, err := tx.Exec(pq.Query, pq.Args...)
	if err != nil {
		return res, err
	}

	return res, nil
}

// RunQuery - reads sql into a struct
func (u *DbUtils) RunQuery(pq *PreparedQuery, dest interface{}) error {
	scanHelper := SQLScan{}
	found := false

	rows, err := u.db.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		found = true
		err = scanHelper.Scan(u, rows, dest)
		break
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	if !found {
		return sql.ErrNoRows
	}

	return nil
}

// RunQueryTx - reads sql into a struct (from a transaction)
func (u *DbUtils) RunQueryTx(tx *sql.Tx, pq *PreparedQuery, dest interface{}) error {
	scanHelper := SQLScan{}
	found := false

	rows, err := tx.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		found = true
		err = scanHelper.Scan(u, rows, dest)
		break
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	if !found {
		return sql.ErrNoRows
	}

	return nil
}

// DBRowCallback - callback type
type DBRowCallback func(row *sql.Rows, sc *SQLScan) error

// ForEachRow - reads sql and runs a function fo every row
func (u *DbUtils) ForEachRow(pq *PreparedQuery, callback DBRowCallback) error {
	sc := new(SQLScan)

	rows, err := u.db.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = callback(rows, sc)
		if err != nil {
			return err
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

// ForEachRowTx - reads sql and runs a function fo every row
func (u *DbUtils) ForEachRowTx(tx *sql.Tx, pq *PreparedQuery, callback DBRowCallback) error {
	sc := new(SQLScan)

	rows, err := tx.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = callback(rows, sc)
		if err != nil {
			return err
		}
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

// GetAllRows - Get all rows
func (u *DbUtils) GetAllRows(pq *PreparedQuery, dest interface{}) error {
	if dest == nil {
		return errors.New("destination must be not null")
	}

	destination := reflect.ValueOf(dest)

	if destination.Kind() != reflect.Slice {
		return errors.New("destination must be an array")
	}

	if destination.IsNil() {
		return errors.New("destination array must be initialised")
	}

	dslice := reflect.Indirect(destination)
	destType := reflect.TypeOf(dest).Elem()
	destKind := destType.Kind()
	isPtr := destKind == reflect.Ptr
	var baseType reflect.Type

	if isPtr {
		baseType = destType.Elem()
	} else {
		baseType = destType
	}

	var err error
	err = u.ForEachRow(pq, func(row *sql.Rows, sc *SQLScan) error {
		destValPtr := reflect.New(baseType)
		val := reflect.Indirect(destValPtr)

		err = sc.Scan(u, row, &val)
		if err != nil {
			return err
		}

		if isPtr {
			dslice.Set(reflect.Append(dslice, destValPtr))
		} else {
			dslice.Set(reflect.Append(dslice, val))
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
