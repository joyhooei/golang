package cls

import (
	"yf_pkg/cachedb"
	"yf_pkg/log"
	"yf_pkg/mysql"
	mysqldb "yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/common"
)

const (
	MODE_TEST       = "test"
	MODE_PRODUCTION = "production"
	MODE_DEVELOP    = "develop"
)

type CustomEnv struct {
	MainDB           *mysql.MysqlDB
	MainRds          *redis.RedisPool
	CacheRds         *redis.RedisPool
	CacheDB          *cachedb.CacheDB
	SortDB           *mysql.MysqlDB
	MsgDB            *mysql.MysqlDB
	StatDB           *mysql.MysqlDB
	DStatDB          *mysql.MysqlDB
	MainLog          *log.MLogger
	Mode             string //运行环境：测试环境/生产环境/开发环境
	UploadServiceUrl string //上传图片和检测图片服务url
	WebServiceUrl    string //web服务URL
}

func (c *CustomEnv) Init(conf *common.Config) (err error) {
	//创建mysql连接池
	c.MainDB, err = mysqldb.New(conf.Mysql.Main.Master, conf.Mysql.Main.Slave)
	if err != nil {
		return err
	}
	c.SortDB, err = mysqldb.New(conf.Mysql.Sort.Master, conf.Mysql.Sort.Slave)
	if err != nil {
		return err
	}
	c.MsgDB, err = mysqldb.New(conf.Mysql.Message.Master, conf.Mysql.Message.Slave)
	if err != nil {
		return err
	}
	c.StatDB, err = mysqldb.New(conf.Mysql.Stat.Master, conf.Mysql.Stat.Slave)
	if err != nil {
		return err
	}
	c.DStatDB, err = mysqldb.New(conf.Mysql.DStat.Master, conf.Mysql.DStat.Slave)
	if err != nil {
		return err
	}
	c.MainRds = redis.New(conf.Redis.Main.Master.String(), conf.Redis.Main.Slave.StringSlice(), conf.Redis.Main.MaxConn)
	c.CacheRds = redis.New(conf.Redis.Cache.Master.String(), conf.Redis.Cache.Slave.StringSlice(), conf.Redis.Cache.MaxConn)

	c.CacheDB = cachedb.New(c.MainDB, c.CacheRds, redis_db.CACHE_DB)
	c.MainLog, err = log.NewMLogger(conf.Log.Dir+"/main", 10000, conf.Log.Level)
	c.Mode = conf.Mode
	c.UploadServiceUrl = conf.UploadServiceUrl
	c.WebServiceUrl = conf.WebServiceUrl
	return err
}
func (c *CustomEnv) GetEnv(module string) *service.Env {
	return service.NewEnv(c)
}
