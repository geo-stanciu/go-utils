package utils

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
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

// DbUtils can be used to prepare queries by changing the sql param notations
// as defined by each supported database
type DbUtils struct {
	sync.RWMutex
	db     *sql.DB
	dbType string
	prefix string
}

func (u *DbUtils) setDbType(dbType string) {
	u.RLock()
	defer u.RUnlock()

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

	if u.dbType == Postgres {
		u.prefix = "$"
	} else if u.dbType == Oracle {
		u.prefix = ":"
	} else {
		u.prefix = ""
	}
}

// PQuery prepares query for run by changing params written as ? to $1, $2, etc
// for postgres and :1, :2, etc for oracle
func (u *DbUtils) PQuery(query string) string {
	q := query
	i := 1

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

	if u.dbType == Postgres {
		q = strings.Replace(q, "current_timestamp", "current_timestamp at time zone 'UTC'", -1)
	} else if u.dbType == MySQL {
		q = strings.Replace(q, "current_timestamp", "UTC_TIMESTAMP()", -1)
	} else if u.dbType == SQLServer {
		q = strings.Replace(q, "getdate()", "getutcdate()", -1)
		q = strings.Replace(q, "current_timestamp", "getutcdate()", -1)
		q = strings.Replace(q, "DATE ?", "convert(date, ?)", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "convert(datetime, ?)", -1)
	} else if u.dbType == Sqlite {
		q = strings.Replace(q, "DATE ?", "date(?)", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "datetime(?)", -1)
	} else if u.dbType == Oracle {
		q = strings.Replace(q, "sysdate", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "current_timestamp", "sys_extract_utc(systimestamp)", -1)
	}

	return q
}

// Connect2Database - connect to a database
func (u *DbUtils) Connect2Database(db **sql.DB, dbType, dbURL string) error {
	u.RLock()
	defer u.RUnlock()

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

// RunQuery - reads sql into a struct
func (u *DbUtils) RunQuery(query string, dest interface{}, args ...interface{}) error {
	scanHelper := SQLScanHelper{}
	found := false

	rows, err := u.db.Query(query, args...)
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
func (u *DbUtils) RunQueryTx(tx *sql.Tx, query string, dest interface{}, args ...interface{}) error {
	scanHelper := SQLScanHelper{}
	found := false

	rows, err := tx.Query(query, args...)
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
func (u *DbUtils) ForEachRow(query string, callback DBRowCallback, args ...interface{}) error {
	rows, err := u.db.Query(query, args...)
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
func (u *DbUtils) ForEachRowTx(tx *sql.Tx, query string, callback DBRowCallback, args ...interface{}) error {
	rows, err := tx.Query(query, args...)
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
