package building

import (
	"errors"
	"fmt"
	"yf_pkg/cachedb"
	"yf_pkg/mysql"
	"yf_pkg/utils"
)

type Building struct {
	Id       string  `json:"id"`
	Name     string  `json:"name"`
	Address  string  `json:"address"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Distence float64 `json:"distence"`
}
type BuildingItems []*Building

func (i BuildingItems) Len() int {
	return len(i)
}

func (items BuildingItems) Less(i, j int) bool {
	return items[i].Distence < items[j].Distence
}

func (items BuildingItems) Swap(i, j int) {
	items[i], items[j] = items[j], items[i]
}
func (h *BuildingItems) Push(x interface{}) {
	*h = append(*h, x.(*Building))
}

func (h *BuildingItems) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

//返回对象的ID，如果没有设置ID，则第二个返回值为false
func (b *Building) ID() (interface{}, bool) {
	return b.Id, b.Id != ""
}

func NewBuilding(id interface{}) cachedb.DBObject {
	return &Building{Id: utils.ToString(id)}
}

//缓存超时时间(秒)，-1表示不超时，0表示默认值10分钟
func (b *Building) Expire() int {
	return 86400
}

//新增或更新数据
func (b *Building) Save(mysqldb *mysql.MysqlDB) (id interface{}, e error) {
	return nil, errors.New("not implemented")
}

//获取数据内容
func (b *Building) Get(id interface{}, mysqldb *mysql.MysqlDB) (e error) {
	b.Id = fmt.Sprintf("%v", id)
	if e = mysqldb.QueryRow("select name,address,lat,lng from building where placeid=?", id).Scan(&b.Name, &b.Address, &b.Lat, &b.Lng); e != nil {
		return e
	}
	return
}

//批量从数据库取数据
func (b *Building) GetMap(ids []interface{}, mysqldb *mysql.MysqlDB) (objs map[interface{}]cachedb.DBObject, e error) {
	rows, e := mysqldb.Query("select placeid,name,address,lat,lng from building where placeid" + mysql.In(ids))
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	objs = make(map[interface{}]cachedb.DBObject)
	for rows.Next() {
		var building Building
		if e = rows.Scan(&building.Id, &building.Name, &building.Address, &building.Lat, &building.Lng); e != nil {
			return nil, e
		}
		objs[building.Id] = &building
	}
	return
}
