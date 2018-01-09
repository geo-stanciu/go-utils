## Goals

Provides easier acces to databases and logging to database.
Tries to implement as much database abstraction as posible.
Allows you to use ? as parameter placeholder in: oracle 12.1, sql server 2017, postgresql, mariadb and mysql.

## Examples

For usage examples, look at: https://github.com/geo-stanciu/go-tryouts/tree/master/go-website

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## Download
```
go get "github.com/geo-stanciu/go-utils/utils"
go get "github.com/sirupsen/logrus"
```

## Usage:

### Declare as vars:
```golang
var (
    log                 = logrus.New()
    audit               = utils.AuditLog{}
    dbUtils             *utils.DbUtils{}
    db                  *sql.DB
)
```

### initialize:
```golang
func init() {
    // Log as JSON instead of the default ASCII formatter.
    log.Formatter = new(logrus.JSONFormatter)
    log.Level = logrus.DebugLevel

    // init databaseutils
    dbUtils = new(utils.DbUtils)
}
```

### in main:

```golang
var err error
var wg sync.WaitGroup

// connect to the database:
err = dbUtils.Connect2Database(&db, "dbtype", "dburl")
if err != nil {
    log.Println(err)
    return
}
defer db.Close()
```

```golang
// setup logger
audit.SetLogger("appname", log, &dbUtils)
audit.SetWaitGroup(&wg)

mw := io.MultiWriter(os.Stdout, audit)
log.Out = mw
```

### Have fun

have fun using db, dbutils and logger
declare each query as:

```golang
pq := dbUtils.PQuery("select count(*) c1 from table1")

pq2 := dbUtils.PQuery(`
    select col1
      from table1
     where col2 = ?
       and col3 = ?
`, val2,
   val3)

pq3 := dbUtils.PQuery("update table1 set col1 = ? where col2 = ?", val1, val2)
```

### Execute Queries

Execute queries with one of:
- Exec - for DML queries (insert, update, delete)
- ExecTx - for DML queries (insert, update, delete)
         - tx is a transaction - type *sql.Tx
- RunQuery - for single row queries
- RunQueryTx - for single row queries
             - tx is a transaction - type *sql.Tx
- dbUtils.ForEachRow,
- dbUtils.ForEachRowTx (where tx is a transaction - type *sql.Tx)
- or standard Exec, Query and QueryRow methods of the database/sql package

```golang
var err error
pq := dbUtils.PQuery(`
    INSERT INTO role (role) VALUES (?)
`, r.Rolename)

_, err = dbUtils.Exec(pq)
if err != nil {
    return err
}
```

```golang
var err error
pq := dbUtils.PQuery(`
    INSERT INTO role (role) VALUES (?)
`, r.Rolename)

_, err = dbUtils.ExecTx(tx, pq)
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
pq := dbUtils.PQuery(`
    SELECT role_id,
            role
        FROM role
        WHERE role_id = ?
`, roleID)

err := dbUtils.RunQuery(pq, r)

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
pq := dbUtils.PQuery(`
    SELECT role_id,
            role
        FROM role
        WHERE role_id = ?
`, roleID)

err := dbUtils.RunQueryTx(tx, pq, r)

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
err = dbUtils.ForEachRow(pq, func(row *sql.Rows) {
    var r MembershipRole
    err = row.Scan(&r.RoleID, &r.Rolename)
    if err != nil {
        return
    }

    roles = append(roles, r)
})
```

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var roles []MembershipRole
var err error
err = dbUtils.ForEachRowTx(tx, pq, func(row *sql.Rows) {
    var r MembershipRole
    err = row.Scan(&r.RoleID, &r.Rolename)
    if err != nil {
        return
    }

    roles = append(roles, r)
})
```

## Use a column scanner

Using a scanner (matches sql with struct columns)
Columns in struct must be declared with "sql" tags
Tags must be columns in the sql query

```golang
type MembershipRole struct {
    RoleID   int    `sql:"role_id"`
    Rolename string `sql:"role"`
}

var roles []MembershipRole
var err error
sc := utils.SQLScanHelper{}
err = dbUtils.ForEachRow(pq, func(row *sql.Rows) {
    var r models.Rate
    err = sc.Scan(&dbUtils, row, &r)
    if err != nil {
        return
    }

    lres.Rates = append(lres.Rates, &r)
})
```

```golang
var roleID int

pq = dbUtils.PQuery(`
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
rows, err := db.Query("select id, name from foo")
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
