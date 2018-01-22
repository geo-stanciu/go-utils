package utils

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// PreparedQuery - prepared query and parameters
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
type PreparedQuery struct {
	DbType      string
	ParamPrefix string
	Query       string
	Args        []interface{}
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

// Prepare - prepares query for running
func (pq *PreparedQuery) Prepare() {
	switch {
	case pq.DbType == Postgres:
		pq.modifyQuery4Postgres()

	case pq.DbType == MySQL:
		pq.modifyQuery4MySQL()

	case pq.DbType == SQLServer:
		pq.modifyQuery4MSSQL()

	case pq.DbType == Oracle || pq.DbType == Oci8:
		pq.modifyQuery4Oracle12c()

	case pq.DbType == Oracle11g:
		pq.modifyQuery4Oracle11g()
	}

	pq.replaceParamPlaceHolders()
}

func (pq *PreparedQuery) modifyQuery4Postgres() {
	q := pq.Query

	q = strings.Replace(q, "now()", "now() at time zone 'UTC'", -1)
	q = strings.Replace(q, "current_timestamp", "current_timestamp at time zone 'UTC'", -1)
	q = strings.Replace(q, "DATE ?", "?", -1)
	q = strings.Replace(q, "TIMESTAMP ?", "?", -1)
	q = strings.Replace(q, "date ?", "?", -1)
	q = strings.Replace(q, "timestamp ?", "?", -1)

	pq.Query = q

	pq.minus2except(true)
	pq.minus2except(false)
}

func (pq *PreparedQuery) modifyQuery4MySQL() {
	q := pq.Query

	backquote := `` + "`" + ``
	q = strings.Replace(q, "now()", "UTC_TIMESTAMP()", -1)
	q = strings.Replace(q, "current_timestamp", "UTC_TIMESTAMP()", -1)
	q = strings.Replace(q, "DATE ?", "?", -1)
	q = strings.Replace(q, "TIMESTAMP ?", "?", -1)
	q = strings.Replace(q, "date ?", "?", -1)
	q = strings.Replace(q, "timestamp ?", "?", -1)
	q = strings.Replace(q, `"`, backquote, -1)

	pq.Query = q

	// Geo
	// MySQL does not support except or minus queries at this time
	// left this here for MariaBD 10.3 who will support EXCEPT
	pq.minus2except(true)
	pq.minus2except(false)
}

func (pq *PreparedQuery) modifyQuery4MSSQL() {
	q := pq.Query

	q = strings.Replace(q, "now()", "getutcdate()", -1)
	q = strings.Replace(q, "getdate()", "getutcdate()", -1)
	q = strings.Replace(q, "current_timestamp", "getutcdate()", -1)
	q = strings.Replace(q, "DATE ?", "convert(date, ?)", -1)
	q = strings.Replace(q, "TIMESTAMP ?", "convert(datetime, ?)", -1)
	q = strings.Replace(q, "date ?", "convert(date, ?)", -1)
	q = strings.Replace(q, "timestamp ?", "convert(datetime, ?)", -1)

	pq.Query = q

	pq.minus2except(true)
	pq.minus2except(false)
	pq.mssqlLimitAndOffset()
}

func (pq *PreparedQuery) modifyQuery4Oracle12c() {
	q := pq.Query

	q = strings.Replace(q, "now()", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "systimestamp", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "sysdate", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "current_timestamp", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "DATE ?", "to_date(?, 'yyyy-mm-dd')", -1)
	q = strings.Replace(q, "TIMESTAMP ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)
	q = strings.Replace(q, "date ?", "to_date(?, 'yyyy-mm-dd')", -1)
	q = strings.Replace(q, "timestamp ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)

	pq.Query = q

	pq.except2minus(true)
	pq.except2minus(false)
	pq.oracle12cLimitAndOffset()
}

func (pq *PreparedQuery) modifyQuery4Oracle11g() {
	q := pq.Query

	q = strings.Replace(q, "now()", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "systimestamp", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "sysdate", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "current_timestamp", "sys_extract_utc(systimestamp)", -1)
	q = strings.Replace(q, "DATE ?", "to_date(?, 'yyyy-mm-dd')", -1)
	q = strings.Replace(q, "TIMESTAMP ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)
	q = strings.Replace(q, "date ?", "to_date(?, 'yyyy-mm-dd')", -1)
	q = strings.Replace(q, "timestamp ?", "to_timestamp(?, 'yyyy-mm-dd HH:mm:ss')", -1)

	pq.Query = q

	pq.except2minus(true)
	pq.except2minus(false)
	pq.oracle11gLimitAndOffset()
}

func (pq *PreparedQuery) replaceParamPlaceHolders() {
	i := 1
	pos := 0
	idx := -1
	var qbuf bytes.Buffer

	idx = strings.Index(pq.Query[pos:], "?")
	if idx < 0 || len(pq.ParamPrefix) == 0 {
		return
	}

	for {
		if idx < 0 {
			qbuf.WriteString(pq.Query[pos:])
			break
		} else {
			qbuf.WriteString(pq.Query[pos : pos+idx])
			pos += idx + 1
		}

		prm := fmt.Sprintf("%s%d", pq.ParamPrefix, i)
		i++

		qbuf.WriteString(prm)
		idx = strings.Index(pq.Query[pos:], "?")
	}

	pq.Query = qbuf.String()
}

func (pq *PreparedQuery) minus2except(searchUppercase bool) {
	pos := 0
	idx := -1
	pos2 := 0
	var qbuf bytes.Buffer

	if searchUppercase {
		idx = strings.Index(pq.Query[pos:], "MINUS")
	} else {
		idx = strings.Index(pq.Query[pos:], "minus")
	}

	if idx < 0 {
		return
	}

	for {
		if idx < 0 {
			qbuf.WriteString(pq.Query[pos:])
			break
		} else {
			qbuf.WriteString(pq.Query[pos : pos+idx])
			pos += idx + len("minus")
		}

		pos2 = pos - len("minus") - 1
		if !IsWhiteSpace(pq.Query[pos:pos+1]) || !IsWhiteSpace(pq.Query[pos2:pos2+1]) {
			qbuf.WriteString(pq.Query[pos2+1 : pos])
		} else {
			if searchUppercase {
				qbuf.WriteString("EXCEPT")
			} else {
				qbuf.WriteString("except")
			}
		}

		if searchUppercase {
			idx = strings.Index(pq.Query[pos:], "MINUS")
		} else {
			idx = strings.Index(pq.Query[pos:], "minus")
		}
	}

	pq.Query = qbuf.String()
}

func (pq *PreparedQuery) except2minus(searchUppercase bool) {
	pos := 0
	idx := -1
	pos2 := 0
	var qbuf bytes.Buffer

	if searchUppercase {
		idx = strings.Index(pq.Query[pos:], "EXCEPT")
	} else {
		idx = strings.Index(pq.Query[pos:], "except")
	}

	if idx < 0 {
		return
	}

	for {
		if idx < 0 {
			qbuf.WriteString(pq.Query[pos:])
			break
		} else {
			qbuf.WriteString(pq.Query[pos : pos+idx])
			pos += idx + len("except")
		}

		pos2 = pos - len("except") - 1
		if !IsWhiteSpace(pq.Query[pos:pos+1]) || !IsWhiteSpace(pq.Query[pos2:pos2+1]) {
			qbuf.WriteString(pq.Query[pos2+1 : pos])
		} else {
			if searchUppercase {
				qbuf.WriteString("MINUS")
			} else {
				qbuf.WriteString("minus")
			}
		}

		if searchUppercase {
			idx = strings.Index(pq.Query[pos:], "EXCEPT")
		} else {
			idx = strings.Index(pq.Query[pos:], "except")
		}
	}

	pq.Query = qbuf.String()
}

func (pq *PreparedQuery) mssqlLimitAndOffset() {
	idx1 := strings.Index(pq.Query, "LIMIT ?")
	idx2 := strings.Index(pq.Query, "OFFSET ?")
	offsetLwCase := false

	if idx1 < 0 {
		idx1 = strings.Index(pq.Query, "limit ?")
	}

	if idx2 < 0 {
		idx2 = strings.Index(pq.Query, "offset ?")
		offsetLwCase = true
	}

	if idx1 > -1 {
		if idx2 > -1 {
			idx3 := idx1 + len("LIMIT ?")
			idx4 := idx2 + len("OFFSET ?")
			q1 := pq.Query[:idx1]
			q2 := pq.Query[idx3:idx2]
			q3 := pq.Query[idx4:]

			pq.Query = fmt.Sprintf("%sOFFSET ? ROWS%sFETCH NEXT ? ROWS ONLY%s", q1, q2, q3)

			if pq.Args != nil {
				n := len(pq.Args)
				if n >= 2 {
					pq.Args = append(pq.Args[:n-2], pq.Args[n-1], pq.Args[n-2])
				}
			}
		} else {
			idx3 := idx1 + len("LIMIT ?")
			q1 := pq.Query[:idx1]
			q3 := pq.Query[idx3:]

			pq.Query = fmt.Sprintf("%sOFFSET 0 ROWS\nFETCH NEXT ? ROWS ONLY%s", q1, q3)
		}
	} else if idx2 > -1 {
		if offsetLwCase {
			pq.Query = strings.Replace(pq.Query, "offset ?", "OFFSET ? ROWS", -1)
		} else {
			pq.Query = strings.Replace(pq.Query, "OFFSET ?", "OFFSET ? ROWS", -1)
		}
	}
}

func (pq *PreparedQuery) oracle12cLimitAndOffset() {
	idx1 := strings.Index(pq.Query, "LIMIT ?")
	idx2 := strings.Index(pq.Query, "OFFSET ?")
	offsetLwCase := false

	if idx1 < 0 {
		idx1 = strings.Index(pq.Query, "limit ?")
	}

	if idx2 < 0 {
		idx2 = strings.Index(pq.Query, "offset ?")
		offsetLwCase = true
	}

	if idx1 > -1 {
		if idx2 > -1 {
			idx3 := idx1 + len("LIMIT ?")
			idx4 := idx2 + len("OFFSET ?")
			q1 := pq.Query[:idx1]
			q2 := pq.Query[idx3:idx2]
			q3 := pq.Query[idx4:]

			pq.Query = fmt.Sprintf("%sOFFSET ? ROWS%sFETCH NEXT ? ROWS ONLY%s", q1, q2, q3)

			if pq.Args != nil {
				n := len(pq.Args)
				if n >= 2 {
					pq.Args = append(pq.Args[:n-2], pq.Args[n-1], pq.Args[n-2])
				}
			}
		} else {
			idx3 := idx1 + len("LIMIT ?")
			q1 := pq.Query[:idx1]
			q3 := pq.Query[idx3:]

			pq.Query = fmt.Sprintf("%sOFFSET 0 ROWS\nFETCH NEXT ? ROWS ONLY%s", q1, q3)
		}
	} else if idx2 > -1 {
		if offsetLwCase {
			pq.Query = strings.Replace(pq.Query, "offset ?", "OFFSET ? ROWS", -1)
		} else {
			pq.Query = strings.Replace(pq.Query, "OFFSET ?", "OFFSET ? ROWS", -1)
		}
	}
}

func (pq *PreparedQuery) oracle11gLimitAndOffset() {
	idx1 := strings.Index(pq.Query, "LIMIT ?")
	idx2 := strings.Index(pq.Query, "OFFSET ?")

	if idx1 < 0 {
		idx1 = strings.Index(pq.Query, "limit ?")
	}

	if idx2 < 0 {
		idx2 = strings.Index(pq.Query, "offset ?")
	}

	if idx1 > -1 {
		q1 := strings.TrimSpace(pq.Query[:idx1])

		if idx2 > -1 {
			pq.Query = fmt.Sprintf("SELECT * FROM (\n%s)\nWHERE rownum BETWEEN ? AND ?", q1)

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
			pq.Query = fmt.Sprintf("SELECT * FROM (\n%s)\nWHERE rownum BETWEEN 0 AND ?", q1)
		}
	} else if idx2 > -1 {
		q1 := strings.TrimSpace(pq.Query[:idx2])

		pq.Query = fmt.Sprintf("SELECT * FROM (\n%s)\nWHERE rownum >= ?", q1)

		if pq.Args != nil {
			n := len(pq.Args)
			if n >= 1 {
				offset := pq.Args[n-1].(int)
				pq.Args[n-1] = offset + 1
			}
		}
	}
}
