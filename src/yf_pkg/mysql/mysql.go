package mysql

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MysqlDB struct {
	rDBs    []*sql.DB
	wDB     *sql.DB
	allRDBs []*sql.DB
	names   map[*sql.DB]string
}

var ErrNoRows = sql.ErrNoRows

func createDB(connStr string) (db *sql.DB, err error) {
	//初始化数据库
	dsn := fmt.Sprintf("%s?charset=utf8&parseTime=false&loc=Asia%%2FShanghai", connStr)
	if db, err = sql.Open("mysql", dsn); err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func New(wConn string, rConns []string) (mdb *MysqlDB, err error) {
	mdb = &MysqlDB{[]*sql.DB{}, nil, []*sql.DB{}, map[*sql.DB]string{}}
	mdb.wDB, err = createDB(wConn)
	if err != nil {
		return nil, err
	}
	for _, rConn := range rConns {
		rdb, err := createDB(rConn)
		if err != nil {
			return nil, err
		}
		mdb.rDBs = append(mdb.rDBs, rdb)
		mdb.allRDBs = append(mdb.allRDBs, rdb)
		mdb.names[rdb] = rConn
	}
	go mdb.checkAlive()
	return
}

func (db *MysqlDB) checkAlive() {
	for {
		time.Sleep(5 * time.Second)
		tmp := make([]*sql.DB, 0, len(db.allRDBs))
		needPrint := false
		status := "db status: "
		for _, rdb := range db.allRDBs {
			if err := rdb.Ping(); err == nil {
				tmp = append(tmp, rdb)
				status += fmt.Sprintf("%v[alive] ", db.names[rdb])
			} else {
				needPrint = true
				status += fmt.Sprintf("%v[down] ", db.names[rdb])
			}
		}
		if needPrint {
			fmt.Println(status)
		}
		db.rDBs = tmp
	}
}

func (db *MysqlDB) Begin() (*sql.Tx, error) {
	return db.wDB.Begin()
}
func (db *MysqlDB) Close() error {
	eStr := ""
	err := db.wDB.Close()
	if err != nil {
		eStr += "close wDB error : " + err.Error() + "\n"
	}
	for i, rdb := range db.rDBs {
		err := rdb.Close()
		if err != nil {
			eStr += fmt.Sprintf("close rDB[%v] error : %v\n", i, err.Error())
		}
	}
	return errors.New(eStr)
}
func (db *MysqlDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.wDB.Exec(query, args...)
}
func (db *MysqlDB) PrepareQuery(query string) (*sql.Stmt, error) {
	tmp := db.rDBs
	if len(tmp) == 0 {
		return nil, errors.New("no available rdbs")
	}
	return db.rDBs[rand.Int()%len(tmp)].Prepare(query)
}

func (db *MysqlDB) Ping() (err error) {
	if err = db.wDB.Ping(); err != nil {
		return err
	}
	for _, rdb := range db.allRDBs {
		if err = rdb.Ping(); err != nil {
			return err
		}
	}
	return nil
}

func (db *MysqlDB) PrepareExec(query string) (*sql.Stmt, error) {
	return db.wDB.Prepare(query)
}
func (db *MysqlDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	tmp := db.rDBs
	if len(tmp) == 0 {
		return nil, errors.New("no available rdbs")
	}
	return db.rDBs[rand.Int()%len(tmp)].Query(query, args...)

}

// 从主库读
func (db *MysqlDB) QueryFromMain(query string, args ...interface{}) (*sql.Rows, error) {
	return db.wDB.Query(query, args...)
}

func (db *MysqlDB) QueryRow(query string, args ...interface{}) *sql.Row {
	tmp := db.rDBs
	if len(tmp) == 0 {
		return db.allRDBs[rand.Int()%len(db.allRDBs)].QueryRow(query, args...)
	}
	return db.rDBs[rand.Int()%len(tmp)].QueryRow(query, args...)
}

// 从主库中查询
func (db *MysqlDB) QueryRowFromMain(query string, args ...interface{}) *sql.Row {
	return db.wDB.QueryRow(query, args...)
}

func In(keys interface{}) string {
	buf := bytes.Buffer{}
	switch ids := keys.(type) {
	case []interface{}:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in ('%v'", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",'%v'", ids[i]))
		}
	case []int:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in (%v", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",%v", ids[i]))
		}
	case []uint32:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in (%v", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",%v", ids[i]))
		}
	case []int32:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in (%v", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",%v", ids[i]))
		}
	case []int64:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in (%v", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",%v", ids[i]))
		}
	case []uint64:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in (%v", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",%v", ids[i]))
		}
	case []string:
		if len(ids) == 0 {
			return " in ('')"
		}
		buf.WriteString(fmt.Sprintf(" in ('%v'", ids[0]))
		for i := 1; i < len(ids); i++ {
			buf.WriteString(fmt.Sprintf(",'%v'", ids[i]))
		}
	}
	buf.WriteString(")")
	return buf.String()
}
