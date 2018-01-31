## Goals

Provides easier acces to databases and logging to database.
Tries to implement as much database abstraction as posible.
Allows you to use ? as parameter placeholder in: oracle 12.1, sql server 2017, postgresql, mariadb and mysql.

## Examples

For usage examples, look at: https://github.com/geo-stanciu/go-web-app

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

Have fun using db, dbutils and logger.
Declare each query as:

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
- dbUtils.**Exec** - for DML queries (insert, update, delete)
- dbUtils.**ExecTx** - for DML queries (insert, update, delete)
         - tx is a transaction - type *sql.Tx
- dbUtils.**RunQuery** - for single row queries
- dbUtils.**RunQueryTx** - for single row queries
             - tx is a transaction - type *sql.Tx
- dbUtils.**ForEachRow**,
- dbUtils.**ForEachRowTx** (where tx is a transaction - type *sql.Tx)
- or standard **Exec**, **Query** and **QueryRow** methods of the database/sql package

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

r := new(MembershipRole)
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

r := new(MembershipRole)
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
err = dbUtils.ForEachRow(pq, func(row *sql.Rows, sc *utils.SQLScan) {
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
err = dbUtils.ForEachRowTx(tx, pq, func(row *sql.Rows, sc *utils.SQLScan) error {
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
err = dbUtils.ForEachRow(pq, func(row *sql.Rows, sc *utils.SQLScan) error {
    r := new(MembershipRole)
    err = sc.Scan(&dbUtils, row, r)
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
pq = dbUtils.PQuery("select id, name from foo")

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
