package utils

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// Postgres - defines PostgreSQL sql driver name
	Postgres string = "postgres"
	// Oracle - defines Oracle sql driver name
	Oracle string = "oci8"
	// Sqlite - defines Sqlite3 driver name
	Sqlite string = "sqlite3"
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

// PreparedQuery - prepared query and parameters
type PreparedQuery struct {
	Query string
	Args  []interface{}
}

// SetArg - Set Arg Value
func (pq *PreparedQuery) SetArg(i int, val interface{}) {
	if i < 0 {
		panic(errors.New("invalid index argument"))
	}

	if pq.Args == nil || len(pq.Args) < i {
		l := len(pq.Args)
		for k := 0; k < l; k++ {
			pq.Args = append(pq.Args, nil)
		}
	}

	pq.Args[i] = val
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
		Oracle,
		Sqlite,
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
	case Oracle:
		u.prefix = ":"
	default:
		u.prefix = ""
	}
}

// PQuery prepares query for run by changing params written as ? to $1, $2, etc
// for postgres and :1, :2, etc for oracle.
// Also, some alterations to the query will be made:
//     - get dates as UTC
//     - in MySQL - replaces quote identifiers with backticks
func (u *DbUtils) PQuery(query string, args ...interface{}) *PreparedQuery {
	pq := PreparedQuery{}
	pq.Args = args

	q := query
	i := 1

	switch u.dbType {
	case Postgres:
		q = strings.Replace(q, "now()", "now() at time zone 'UTC'", -1)
		q = strings.Replace(q, "current_timestamp", "current_timestamp at time zone 'UTC'", -1)
		q = strings.Replace(q, "DATE ?", "?", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "?", -1)

	case MySQL:
		backquote := `` + "`" + ``
		q = strings.Replace(q, "now()", "UTC_TIMESTAMP()", -1)
		q = strings.Replace(q, "current_timestamp", "UTC_TIMESTAMP()", -1)
		q = strings.Replace(q, "DATE ?", "?", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "?", -1)
		q = strings.Replace(q, `"`, backquote, -1)

	case SQLServer:
		q = strings.Replace(q, "getdate()", "getutcdate()", -1)
		q = strings.Replace(q, "current_timestamp", "getutcdate()", -1)
		q = strings.Replace(q, "DATE ?", "convert(date, ?)", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "convert(datetime, ?)", -1)

	case Sqlite:
		q = strings.Replace(q, "DATE ?", "date(?)", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "datetime(?)", -1)

	case Oracle:
		q = strings.Replace(q, "systimestamp", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "sysdate", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "current_timestamp", "sys_extract_utc(systimestamp)", -1)
	}

	if len(u.prefix) > 0 {
		for {
			idx := strings.Index(q, "?")

			if idx < 0 {
				break
			}

			prm := fmt.Sprintf("%s%d", u.prefix, i)
			i++

			q = strings.Replace(q, "?", prm, 1)
		}
	}

	pq.Query = q

	return &pq
}

// Connect2Database - connect to a database
func (u *DbUtils) Connect2Database(db **sql.DB, dbType, dbURL string) error {
	var err error
	u.setDbType(dbType)

	*db, err = sql.Open(dbType, dbURL)
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
	scanHelper := SQLScanHelper{}
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
	scanHelper := SQLScanHelper{}
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
type DBRowCallback func(row *sql.Rows)

// ForEachRow - reads sql and runs a function fo every row
func (u *DbUtils) ForEachRow(pq *PreparedQuery, callback DBRowCallback) error {
	rows, err := u.db.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		callback(rows)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

// ForEachRowTx - reads sql and runs a function fo every row
func (u *DbUtils) ForEachRowTx(tx *sql.Tx, pq *PreparedQuery, callback DBRowCallback) error {
	rows, err := tx.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		callback(rows)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}
