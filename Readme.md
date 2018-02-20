# Goals

Provides easier acces to databases and logging to database.
Tries to implement as much database abstraction as posible.
Allows you to use ? as parameter placeholder in: oracle 12.1, sql server 2017, postgresql, mariadb and mysql.

If you need to write the character ? in a query (such as testing if a key exists in a postgresql jsonb column) you must write it as ??

## Examples

For usage examples, look at: https://github.com/geo-stanciu/go-web-app

## Features

- Prepared queries and parameters
- Query parameter placeholders will be written as ? in all suported databses.
- Some alterations to the query will be made:
  - get dates as UTC
  - in Postgresql
    - changes params written as ? to $1, $2, etc
  - in MySQL
    - replaces quote identifiers with backticks
  - in SQL Server
    - replaces "LIMIT ? OFFSET ?" with "OFFSET ? ROWS FETCH NEXT ? ROWS ONLY"
    - switches parameters set for OFFSET and LIMIT to reflect the changed query
  - Limitations:
    - LIMIT ? OFFSET ? must be the last 2 parameters in the query
  - in Oracle
  - changes params written as ? to :1, :2, etc
- Provides an automatic sql column to struct field matcher
  - SQLScan helper class for reading sql to Struct
  Columns in struct must be marked with a `sql:"col_name"` tag
  Ex: in sql a column name is col1, in struct the col tag must be `sql:"col1"`

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## Download

```bash
go get "github.com/geo-stanciu/go-utils/utils"
go get "github.com/sirupsen/logrus"
```

## Usage

### Declare as vars

```golang
var (
    log                 = logrus.New()
    audit               = utils.AuditLog{}
    dbutl               *utils.DbUtils{}
    db                  *sql.DB
)
```

### initialize

```golang
func init() {
    // Log as JSON instead of the default ASCII formatter.
    log.Formatter = new(logrus.JSONFormatter)
    log.Level = logrus.DebugLevel

    // init databaseutils
    dbutl = new(utils.DbUtils)
}
```

### in main

```golang
var err error
var wg sync.WaitGroup

// connect to the database:
err = dbutl.Connect2Database(&db, "dbtype", "dburl")
if err != nil {
    log.Println(err)
    return
}
defer db.Close()
```

```golang
// setup logger
audit.SetLogger("appname", log, dbutl)
audit.SetWaitGroup(&wg)
defer audit.Close()

mw := io.MultiWriter(os.Stdout, audit)
log.Out = mw
```

### Have fun

Have fun using db, dbutl and logger.
Declare each query as:

```golang
pq := dbutl.PQuery("select count(*) c1 from table1")

pq2 := dbutl.PQuery(`
    select col1
      from table1
     where col2 = ?
       and col3 = ?
`, val2,
   val3)

pq3 := dbutl.PQuery("update table1 set col1 = ? where col2 = ?", val1, val2)
```

### Execute Queries

Execute queries with one of:

- dbutl.**Exec** - for DML queries (insert, update, delete)
- dbutl.**ExecTx** - for DML queries (insert, update, delete)
         - tx is a transaction - type *sql.Tx
- dbutl.**RunQuery** - for single row queries
- dbutl.**RunQueryTx** - for single row queries
             - tx is a transaction - type *sql.Tx
- dbutl.**ForEachRow**,
- dbutl.**ForEachRowTx** (where tx is a transaction - type *sql.Tx)
- or standard **Exec**, **Query** and **QueryRow** methods of the database/sql package

```golang
var err error
pq := dbutl.PQuery(`
    INSERT INTO role (role) VALUES (?)
`, r.Rolename)

_, err = dbutl.Exec(pq)
if err != nil {
    return err
}
```

```golang
var err error
pq := dbutl.PQuery(`
    INSERT INTO role (role) VALUES (?)
`, r.Rolename)

_, err = dbutl.ExecTx(tx, pq)
if err != nil {
    return err
}
```

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var err error
pq := dbutl.PQuery(`
    SELECT role_id,
            role
        FROM role
        WHERE role_id = ?
`, roleID)

r := new(MembershipRole)
err := dbutl.RunQuery(pq, r)

switch {
case err == sql.ErrNoRows:
    return fmt.Errorf("role not found")
case err != nil:
    return err
}
```

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var err error
pq := dbutl.PQuery(`
    SELECT role_id,
            role
        FROM role
        WHERE role_id = ?
`, roleID)

r := new(MembershipRole)
err := dbutl.RunQueryTx(tx, pq, r)

switch {
case err == sql.ErrNoRows:
    return fmt.Errorf("role not found")
case err != nil:
    return err
}
```

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var roles []MembershipRole
var err error
err = dbutl.ForEachRow(pq, func(row *sql.Rows, sc *utils.SQLScan) {
    var r MembershipRole
    err = row.Scan(&r.RoleID, &r.Rolename)
    if err != nil {
        return err
    }

    roles = append(roles, r)
    return nil
})
```

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var roles []MembershipRole
var err error
err = dbutl.ForEachRowTx(tx, pq, func(row *sql.Rows, sc *utils.SQLScan) error {
    var r MembershipRole
    err = row.Scan(&r.RoleID, &r.Rolename)
    if err != nil {
        return err
    }

    roles = append(roles, r)
    return nil
})
```

## Use the column scanner

Using a scanner (matches sql with struct columns).
Columns in struct must be declared with "sql" tags.
Tags must be columns in the sql query.

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var roles []MembershipRole
var err error
err = dbutl.ForEachRow(pq, func(row *sql.Rows, sc *utils.SQLScan) error {
    r := new(MembershipRole)
    err = sc.Scan(dbutl, row, r)
    if err != nil {
        return err
    }

    lres.Rates = append(lres.Rates, &r)
    return nil
})
```

## Use standard databse/sql package methods

```golang
var roleID int

pq = dbutl.PQuery(`
    SELECT role_id FROM role WHERE lower(role) = lower(?)
`, r.Rolename)

err = db.QueryRow(pq.Query, pq.Args...).Scan(roleID)

switch {
case err == sql.ErrNoRows:
    r.RoleID = -1
case err != nil:
    return err
}
```

```golang
pq = dbutl.PQuery("select id, name from foo")

rows, err := db.Query(pq.Query, pq.Args...)
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    var id int
    var name string
    err = rows.Scan(&id, &name)
    if err != nil {
        return err
    }

    fmt.Println(id, name)
}

err = rows.Err()
if err != nil {
    return err
}

rows.Close()
```
