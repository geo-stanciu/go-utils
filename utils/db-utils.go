package utils

import (
	"bytes"
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
	// Oracle - defines Oracle sql driver name - for Oracle 12c+
	Oracle string = "oracle"
	// Oci8 - defines Oracle sql driver name - for Oracle 12c+
	Oci8 string = "oci8"
	// Oracle11g - defines Oracle sql driver name - for Oracle11g
	Oracle11g string = "oracle11g"
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

	n := 0
	if pq.Args != nil {
		n = len(pq.Args)
	}

	if n < i {
		for k := 0; k < n; k++ {
			pq.Args = append(pq.Args, nil)
		}
	} else if n == 0 {
		pq.Args = append(pq.Args, nil)
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
		Oci8,
		Oracle,
		Oracle11g,
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
	pq := PreparedQuery{}
	pq.Args = args
	q := query

	switch {
	case u.dbType == Postgres:
		q = strings.Replace(q, "now()", "now() at time zone 'UTC'", -1)
		q = strings.Replace(q, "current_timestamp", "current_timestamp at time zone 'UTC'", -1)
		q = strings.Replace(q, "DATE ?", "?", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "?", -1)
		q = strings.Replace(q, "date ?", "?", -1)
		q = strings.Replace(q, "timestamp ?", "?", -1)

	case u.dbType == MySQL:
		backquote := `` + "`" + ``
		q = strings.Replace(q, "now()", "UTC_TIMESTAMP()", -1)
		q = strings.Replace(q, "current_timestamp", "UTC_TIMESTAMP()", -1)
		q = strings.Replace(q, "DATE ?", "?", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "?", -1)
		q = strings.Replace(q, "date ?", "?", -1)
		q = strings.Replace(q, "timestamp ?", "?", -1)
		q = strings.Replace(q, `"`, backquote, -1)

	case u.dbType == SQLServer:
		q = strings.Replace(q, "getdate()", "getutcdate()", -1)
		q = strings.Replace(q, "current_timestamp", "getutcdate()", -1)
		q = strings.Replace(q, "DATE ?", "convert(date, ?)", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "convert(datetime, ?)", -1)
		q = strings.Replace(q, "date ?", "convert(date, ?)", -1)
		q = strings.Replace(q, "timestamp ?", "convert(datetime, ?)", -1)

		idx1 := strings.Index(q, "LIMIT ?")
		idx2 := strings.Index(q, "OFFSET ?")
		offsetLwCase := false

		if idx1 < 0 {
			idx1 = strings.Index(q, "limit ?")
		}

		if idx2 < 0 {
			idx2 = strings.Index(q, "offset ?")
			offsetLwCase = true
		}

		if idx1 > -1 {
			if idx2 > -1 {
				idx3 := idx1 + len("LIMIT ?")
				idx4 := idx2 + len("OFFSET ?")
				q1 := q[:idx1]
				q2 := q[idx3:idx2]
				q3 := q[idx4:]

				q = fmt.Sprintf("%sOFFSET ? ROWS%sFETCH NEXT ? ROWS ONLY%s", q1, q2, q3)

				if pq.Args != nil {
					n := len(pq.Args)
					if n >= 2 {
						pq.Args = append(pq.Args[:n-2], pq.Args[n-1], pq.Args[n-2])
					}
				}
			} else {
				idx3 := idx1 + len("LIMIT ?")
				q1 := q[:idx1]
				q3 := q[idx3:]

				q = fmt.Sprintf("%sOFFSET 0 ROWS\nFETCH NEXT ? ROWS ONLY%s", q1, q3)
			}
		} else if idx2 > -1 {
			if offsetLwCase {
				q = strings.Replace(q, "offset ?", "OFFSET ? ROWS", -1)
			} else {
				q = strings.Replace(q, "OFFSET ?", "OFFSET ? ROWS", -1)
			}
		}

	case u.dbType == Sqlite:
		q = strings.Replace(q, "DATE ?", "date(?)", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "datetime(?)", -1)
		q = strings.Replace(q, "date ?", "date(?)", -1)
		q = strings.Replace(q, "timestamp ?", "datetime(?)", -1)

	case u.dbType == Oracle || u.dbType == Oci8:
		q = strings.Replace(q, "systimestamp", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "sysdate", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "current_timestamp", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "DATE ?", "to_date(?, 'yyyy-mm-dd')", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)
		q = strings.Replace(q, "date ?", "to_date(?, 'yyyy-mm-dd')", -1)
		q = strings.Replace(q, "timestamp ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)

		idx1 := strings.Index(q, "LIMIT ?")
		idx2 := strings.Index(q, "OFFSET ?")
		offsetLwCase := false

		if idx1 < 0 {
			idx1 = strings.Index(q, "limit ?")
		}

		if idx2 < 0 {
			idx2 = strings.Index(q, "offset ?")
			offsetLwCase = true
		}

		if idx1 > -1 {
			if idx2 > -1 {
				idx3 := idx1 + len("LIMIT ?")
				idx4 := idx2 + len("OFFSET ?")
				q1 := q[:idx1]
				q2 := q[idx3:idx2]
				q3 := q[idx4:]

				q = fmt.Sprintf("%sOFFSET ? ROWS%sFETCH NEXT ? ROWS ONLY%s", q1, q2, q3)

				if pq.Args != nil {
					n := len(pq.Args)
					if n >= 2 {
						pq.Args = append(pq.Args[:n-2], pq.Args[n-1], pq.Args[n-2])
					}
				}
			} else {
				idx3 := idx1 + len("LIMIT ?")
				q1 := q[:idx1]
				q3 := q[idx3:]

				q = fmt.Sprintf("%sOFFSET 0 ROWS\nFETCH NEXT ? ROWS ONLY%s", q1, q3)
			}
		} else if idx2 > -1 {
			if offsetLwCase {
				q = strings.Replace(q, "offset ?", "OFFSET ? ROWS", -1)
			} else {
				q = strings.Replace(q, "OFFSET ?", "OFFSET ? ROWS", -1)
			}
		}

	case u.dbType == Oracle11g:
		q = strings.Replace(q, "systimestamp", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "sysdate", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "current_timestamp", "sys_extract_utc(systimestamp)", -1)
		q = strings.Replace(q, "DATE ?", "to_date(?, 'yyyy-mm-dd')", -1)
		q = strings.Replace(q, "TIMESTAMP ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)
		q = strings.Replace(q, "date ?", "to_date(?, 'yyyy-mm-dd')", -1)
		q = strings.Replace(q, "timestamp ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)

		idx1 := strings.Index(q, "LIMIT ?")
		idx2 := strings.Index(q, "OFFSET ?")

		if idx1 < 0 {
			idx1 = strings.Index(q, "limit ?")
		}

		if idx2 < 0 {
			idx2 = strings.Index(q, "offset ?")
		}

		if idx1 > -1 {
			q1 := strings.TrimSpace(q[:idx1])

			if idx2 > -1 {
				q = fmt.Sprintf("SELECT * FROM (\n%s)\nWHERE rownum BETWEEN ? AND ?", q1)

				if pq.Args != nil {
					n := len(pq.Args)
					if n >= 2 {
						pq.Args = append(pq.Args[:n-2], pq.Args[n-1], pq.Args[n-2])
						offset := pq.Args[n-2].(int)
						nrRows := pq.Args[n-1].(int)
						pq.Args[n-2] = offset + 1
						pq.Args[n-1] = offset + nrRows
					}
				}
			} else {
				q = fmt.Sprintf("SELECT * FROM (\n%s)\nWHERE rownum BETWEEN 0 AND ?", q1)
			}
		} else if idx2 > -1 {
			q1 := strings.TrimSpace(q[:idx2])

			q = fmt.Sprintf("SELECT * FROM (\n%s)\nWHERE rownum >= ?", q1)

			if pq.Args != nil {
				n := len(pq.Args)
				if n >= 1 {
					offset := pq.Args[n-1].(int)
					pq.Args[n-1] = offset + 1
				}
			}
		}
	}

	i := 1
	pos := 0
	idx := -1
	var qbuf bytes.Buffer

	if len(u.prefix) > 0 {
		for {
			idx = strings.Index(q[pos:], "?")

			if idx < 0 {
				qbuf.WriteString(q[pos:])
				break
			} else {
				qbuf.WriteString(q[pos : pos+idx])
				pos += idx + 1
			}

			prm := fmt.Sprintf("%s%d", u.prefix, i)
			i++

			qbuf.WriteString(prm)
		}

		pq.Query = qbuf.String()
	} else {
		pq.Query = q
	}

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
type DBRowCallback func(row *sql.Rows) error

// ForEachRow - reads sql and runs a function fo every row
func (u *DbUtils) ForEachRow(pq *PreparedQuery, callback DBRowCallback) error {
	rows, err := u.db.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = callback(rows)
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
	rows, err := tx.Query(pq.Query, pq.Args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		err = callback(rows)
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
