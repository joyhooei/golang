package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"code.serveyou.cn/common"
	"code.serveyou.cn/location"
	"code.serveyou.cn/pkg/config"
	"code.serveyou.cn/pkg/format"
	"code.serveyou.cn/pkg/log"
)

type City struct {
	Id       uint
	Name     string
	EName    string
	Lat      float32
	Lng      float32
	Province uint8
}
type Province struct {
	Id     uint8
	Name   string
	Domain string
}

var CityMap map[uint]City = map[uint]City{}
var ProvinceMap map[uint8]Province = map[uint8]Province{}

var CitiesJSON format.JSON
var loc *location.Location
var SearchRange uint //城市搜索范围（米）

var glog *log.Logger
var conf config.Config

//配置文件中的必要项
var keywords = map[string]bool{
	"ip":         true,
	"port":       true,
	"log":        true,
	"my_user":    true,
	"my_pwd":     true,
	"my_db":      true,
	"my_ip":      true,
	"my_port":    true,
	"city_range": true,
}

//检查配置文件是否合法
func checkConfig(conf *config.Config) error {
	for key := range keywords {
		if _, found := conf.Items[key]; found == false {
			return errors.New("not found key [" + key + "]")
		}
	}
	return nil
}

func GenJSON() (err error) {
	var db *sql.DB
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", conf.Items["my_user"], conf.Items["my_pwd"],
		conf.Items["my_ip"], conf.Items["my_port"], conf.Items["my_db"])
	if db, err = common.NewDB(dsn); err != nil {
		return
	}
	defer db.Close()

	provinces := make([]Province, 0, 34)
	cities := make(map[uint8][]City)
	sql := "select ID,Name,Domain from Province order by Domain"
	rows, err := db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var province Province
		err = rows.Scan(&province.Id, &province.Name, &province.Domain)
		if err != nil {
			return
		}
		provinces = append(provinces, province)
		ProvinceMap[province.Id] = province
	}
	sql = "select ID,Name,EName,Province,Latitude,Longitude from City order by EName"
	rows, err = db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()
	elements := make([]location.Element, 0, 10)
	for rows.Next() {
		var city City
		err = rows.Scan(&city.Id, &city.Name, &city.EName, &city.Province, &city.Lat, &city.Lng)
		if err != nil {
			return
		}
		elements = append(elements, location.Element{city.Id, city.Lat, city.Lng})
		cl, found := cities[city.Province]
		if !found {
			cl = make([]City, 0, 20)
		}
		cl = append(cl, city)
		cities[city.Province] = cl
		CityMap[city.Id] = city
	}
	loc = location.NewLocation(elements)

	plist := make([]map[string]interface{}, 0, 34)
	for _, p := range provinces {
		pr := make(map[string]interface{})
		pr["id"] = p.Id
		pr["name"] = p.Name
		pr["domain"] = p.Domain
		c, found := cities[p.Id]
		cl := make([]map[string]interface{}, 0, 10)
		if found {
			for _, ci := range c {
				city := make(map[string]interface{})
				city["id"] = ci.Id
				city["name"] = ci.Name
				city["ename"] = ci.EName
				city["lat"] = ci.Lat
				city["lng"] = ci.Lng
				cl = append(cl, city)
			}
		}
		pr["cities"] = cl
		plist = append(plist, pr)
	}
	CitiesJSON = format.GenerateJSON(plist)
	fmt.Println(CitiesJSON)
	return
}

func listCities(r *http.Request, body string, result map[string]interface{}) (err error) {
	fmt.Printf("body=%v\n", body)
	values, err := format.ParseKV(string(body), "\r\n")
	if err != nil {
		return
	}
	latStr, foundLat := values["lat"]
	lngStr, foundLng := values["lng"]
	if foundLat && foundLng {
		lat, err := format.ParseFloat(latStr)
		if err != nil {
			return err
		}
		lng, err := format.ParseFloat(lngStr)
		if err != nil {
			return err
		}
		elems, err := loc.Adjacent2(lat, lng, SearchRange)
		if err != nil {
			return err
		}
		num := 5
		if len(elems) < 5 {
			num = len(elems)
		}
		adjCities := make([]map[string]interface{}, 0, num)
		for i := 0; i < num; i++ {
			cs := make(map[string]interface{})
			ci := CityMap[elems[i].Id]
			cs["id"] = ci.Id
			cs["name"] = ci.Name
			cs["ename"] = ci.EName
			cs["lat"] = ci.Lat
			cs["lng"] = ci.Lng
			cs["distance"] = elems[i].Distance
			cs["pname"] = ProvinceMap[ci.Province].Name
			cs["pid"] = ci.Province
			cs["domain"] = ProvinceMap[ci.Province].Domain
			adjCities = append(adjCities, cs)
		}
		result["adjacent"] = adjCities
	}
	result["all"] = CitiesJSON

	return
}

//普通请求，全部明文
func nonSecureHandler(w http.ResponseWriter, r *http.Request) {
	var result map[string]interface{} = make(map[string]interface{})
	result["status"] = "fail"
	var err error
	cmd := strings.Split(r.URL.Path[1:], "/")
	fmt.Println("cmd = " + cmd[0])

	body_bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		result["error"] = err.Error()
		w.Write([]byte(format.GenerateJSON(result)))
		return
	}
	body := string(body_bytes)
	switch cmd[0] {
	case "list_cities":
		err = listCities(r, body, result)
	default:
		result["error"] = "unknown command : " + r.URL.Path[1:]
		w.Write([]byte(format.GenerateJSON(result)))
		return
	}
	if err == nil {
		result["status"] = "ok"
		w.Write([]byte(format.GenerateJSON(result)))
	} else {
		result["error"] = err.Error()
		w.Write([]byte(format.GenerateJSON(result)))
	}
}
func main() {
	if len(os.Args) < 2 {
		fmt.Printf("invalid args : %s [config]\n", os.Args[0])
		return
	}

	var err error
	conf, err = config.NewConfig(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = checkConfig(&conf)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	glog, err := log.New(conf.Items["log"], 10000)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer glog.Close()
	glog.Append("start service " + os.Args[0])

	if err = GenJSON(); err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}
	SearchRange, err = format.ParseUint(conf.Items["city_range"])
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
		return
	}

	http.HandleFunc("/", nonSecureHandler)
	err = http.ListenAndServe(conf.Items["ip"]+":"+conf.Items["port"], nil)
	if err != nil {
		fmt.Println(err.Error())
		glog.Append(err.Error())
	}
}
