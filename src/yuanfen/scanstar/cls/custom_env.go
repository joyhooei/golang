package cls

import (
	"yf_pkg/log"
	"yf_pkg/mysql"
	mysqldb "yf_pkg/mysql"
	"yf_pkg/service"
	"yuanfen/scanstar/cls/common"
)

type CustomEnv struct {
	MainDB  *mysql.MysqlDB
	StatDB  *mysql.MysqlDB
	MainLog *log.MLogger
}

func (c *CustomEnv) Init(conf *common.Config) (err error) {
	//创建mysql连接池
	c.MainDB, err = mysqldb.New(conf.Mysql.Main.Master, conf.Mysql.Main.Slave)
	if err != nil {
		return err
	}
	c.StatDB, err = mysqldb.New(conf.Mysql.Stat.Master, conf.Mysql.Stat.Slave)
	if err != nil {
		return err
	}

	c.MainLog, err = log.NewMLogger(conf.Log.Dir+"/main", 10000, conf.Log.Level)
	return err
}
func (c *CustomEnv) GetEnv(module string) *service.Env {
	return service.NewEnv(c)
}
