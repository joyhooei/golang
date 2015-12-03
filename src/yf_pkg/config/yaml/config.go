package yaml

import (
	"errors"
	"fmt"
	"net"
	"os"

	"gopkg.in/yaml.v1"
)

type Address struct {
	Ip   string "ip"
	Port int    "port"
}

type Addresses []Address

type MysqlType struct {
	Master string   "master"
	Slave  []string "slave"
}

type RedisType struct {
	Master  Address   "master"
	Slave   Addresses "slave"
	MaxConn int       "max_conn"
}

func (a *Address) String(lookup ...bool) string {
	if a.Ip == "0.0.0.0" {
		return fmt.Sprintf(":%v", a.Port)
	}
	if len(lookup) > 0 && lookup[0] == false {
		return fmt.Sprintf("%s:%v", a.Ip, a.Port)
	}

	ip, e := net.LookupIP(a.Ip)
	if e != nil {
		return fmt.Sprintf("%s:%v", a.Ip, a.Port)
	} else {
		return fmt.Sprintf("%s:%v", ip[0].String(), a.Port)
	}
}

func (as Addresses) StringSlice() []string {
	ret := make([]string, len(as))
	for i, v := range as {
		ret[i] = v.String()
	}
	return ret
}

func Load(c interface{}, path string) error {
	file, e := os.Open(path)
	if e != nil {
		return e
	}
	info, e := file.Stat()
	if e != nil {
		return e
	}
	defer file.Close()
	data := make([]byte, info.Size())
	n, e := file.Read(data)
	if e != nil {
		return e
	}
	if int64(n) < info.Size() {
		return errors.New(fmt.Sprintf("cannot read %v bytes from %v", info.Size(), path))
	}

	e = yaml.Unmarshal([]byte(data), c)
	return e
}
