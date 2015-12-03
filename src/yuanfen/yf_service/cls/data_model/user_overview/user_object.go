package user_overview

import (
	"database/sql"
	"errors"
	"strings"
	"time"
	"yf_pkg/cachedb"
	"yf_pkg/format"
	"yf_pkg/mysql"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
)

type UserViewItem struct {
	Uid              uint32    //uid
	Gender           int       //性别
	Username         string    //慕慕ID
	Isvip            int       //是否VIP 1为VIP
	Grade            int       //等级
	Nickname         string    //昵称
	Avatar           string    //头像
	Province         string    //省
	City             string    //市
	Birthday         time.Time //生日
	Age              int       //年龄
	Height           int       //身高
	Income           int       //收入（上限）
	Star             int       //星座，0-12，0表示未填写
	Aboutme          string    //内心独白
	Avatarlevel      int       //头像好坏级别 -1未通过 0差 3默认 6好 9 优秀
	Ltag             Localtag  //附近标签
	Tag              []string  //[]string  //性格标签 列表
	Stat             int       //用户是否被封 0为正常 5为封号
	CertifyPhone     int       //用户是否手机认证，0： 未认证，1：已认证
	CertifyVideo     int       //用户是否视频认证，0： 未认证，1：已认证
	CertifyIDcard    int       //用户是否身份证认证，0： 未认证，1：已认证
	HonestyLevel     int       //用户诚信等级
	CertifyLevel     int       //用户认证等级
	VideoImg         string    //视频认证头像
	Homeprovince     string    //家乡省份
	Homecity         string    //家乡城市
	Workunit         string    //工作单位
	Trade            string    //行业
	Job              string    //职业
	School           string    //毕业院校
	Edu              int       //学历
	Needtag          []string  //感兴趣的类型 列表
	Require          RequireObj
	WorkPlaceName    string  //工作地点
	WorkPlaceId      string  //工作地点的百度地图ID
	WorkPlaceAddress string  //工作地点的详细地址
	WorkLat          float64 //工作地点 纬度 为0表示没填或者手动输入
	WorkLng          float64 //工作地点 经度 为0表示没填或者手动输入
	Dynamic_img      string  //动态背景图片
}

//是否值得推荐
func (u *UserViewItem) IsRecommend() bool {
	return u.Avatarlevel > common.AVLEVEL_VALID
}

//是否合法
func (u *UserViewItem) IsValid() bool {
	return u.Avatarlevel > common.AVLEVEL_INVALID
}

//是否在审核中
func (u *UserViewItem) IsAuditing() bool {
	return u.Avatarlevel == common.AVLEVEL_PENDING
}

type RequireObj struct {
	Province    string //要求对方所在省 默认值为空 不限制为"不限"
	City        string //要求对方所在市 默认值为空 不限制为"不限"
	Minage      int    //年龄最小为 为0表示无最小限制  -1为默认值
	Maxage      int    //年龄最大 为999表示无限制  -1为默认值
	Minheight   int    //身高最低 0为不限制  -1为默认值
	Maxheight   int    //身高最高 999为不限制 -1为默认值
	Minedu      int    //学历最低 0为不限制  -1为默认值
	Hardrequire int    //是否硬性要求 0非硬性 1 是硬性 2 不全是硬性 默认0
}

//已填写的择友条件数量
func (req *RequireObj) Filled() (num int) {
	if req.Province != "" {
		num++
	}
	if req.Minage != 0 || req.Maxage != common.MAX_AGE {
		num++
	}
	if req.Minheight != 0 || req.Maxheight != common.MAX_HEIGHT {
		num++
	}
	if req.Minedu > 0 {
		num++
	}
	return
}

//检查candidate是否满足req的要求
func (req *RequireObj) Match(candidate *UserViewItem) bool {
	if req.Province != "" && candidate.Province != req.Province {
		return false
	}
	if req.City != "" && candidate.City != req.City {
		return false
	}
	if req.Minage > candidate.Age || req.Maxage < candidate.Age {
		return false
	}
	if req.Minheight > candidate.Height || req.Maxheight < candidate.Height {
		return false
	}
	if req.Minedu <= candidate.Edu {
		return false
	}
	return true
}

func NewUserViewItem(uid interface{}) cachedb.DBObject {
	user := &UserViewItem{}
	switch v := uid.(type) {
	case uint32:
		user.Uid = v
	}
	return user
}

func (u *UserViewItem) ID() (id interface{}, ok bool) {
	return u.Uid, u.Uid != 0
}

func (u *UserViewItem) Save(mysqldb *mysql.MysqlDB) (id interface{}, e error) {
	return nil, errors.New("not implemented")
}

func (u *UserViewItem) Get(id interface{}, mysqldb *mysql.MysqlDB) (e error) {
	var birthday string
	var mtag, tagstr string
	e = mdb.QueryRow("select a.uid,gender,nickname,avatar,province,city,height,aboutme,avatarlevel,tag,stat,star,phonestat,certify_video,homeprovince,homecity,workarea,workunit,job,school,trade,edu,birthday,requireprovince,requirecity,minage,maxage,minheight,maxheight,minedu,needtag,hardrequire,b.workplaceid,IFNULL(building.lat,0),IFNULL(building.lng,0),IFNULL(building.`address`,''),dynamic_img from user_main a LEFT JOIN user_detail b on a.uid=b.uid LEFT JOIN building on building.placeid=b.workplaceid where a.uid=?", id).Scan(&u.Uid, &u.Gender, &u.Nickname, &u.Avatar, &u.Province, &u.City, &u.Height, &u.Aboutme, &u.Avatarlevel, &mtag, &u.Stat, &u.Star, &u.CertifyPhone, &u.CertifyVideo, &u.Homeprovince, &u.Homecity, &u.WorkPlaceName, &u.Workunit, &u.Job, &u.School, &u.Trade, &u.Edu, &birthday, &u.Require.Province, &u.Require.City, &u.Require.Minage, &u.Require.Maxage, &u.Require.Minheight, &u.Require.Maxheight, &u.Require.Minedu, &tagstr, &u.Require.Hardrequire, &u.WorkPlaceId, &u.WorkLat, &u.WorkLng, &u.WorkPlaceAddress, &u.Dynamic_img)
	if e != nil {
		return e
	}
	u.Tag = strings.Split(mtag, ",")
	u.Needtag = strings.Split(tagstr, ",")
	u.Birthday, _ = utils.ToTime(birthday, format.TIME_LAYOUT_1)
	u.Age = utils.BirthdayToAge(u.Birthday)
	u.Ltag = NewLocaltag()
	var tm, timeout string
	e = mdb.QueryRow("select content,gender,tm,timeout,`range`,min_age,max_age,min_height,max_height,star,income,certify_phone,certify_video,certify_idcard from user_tag where uid=? and timeout > ?", id, utils.Now).Scan(&u.Ltag.Content, &u.Ltag.Req.Gender, &tm, &timeout, &u.Ltag.Req.Range, &u.Ltag.Req.MinAge, &u.Ltag.Req.MaxAge, &u.Ltag.Req.MinHeight, &u.Ltag.Req.MaxHeight, &u.Ltag.Req.Star, &u.Ltag.Req.Income, &u.Ltag.Req.CertifyPhone, &u.Ltag.Req.CertifyVideo, &u.Ltag.Req.CertifyIDcard)
	switch e {
	case nil, sql.ErrNoRows:
	default:
		return e
	}
	u.Ltag.Tm, _ = utils.ToTime(tm, format.TIME_LAYOUT_1)
	u.Ltag.Timeout, _ = utils.ToTime(timeout, format.TIME_LAYOUT_1)
	u.Ltag.HasReq = !u.Ltag.Req.NoRequirement()

	return nil
}

func (u *UserViewItem) GetMap(ids []interface{}, mysqldb *mysql.MysqlDB) (objs map[interface{}]cachedb.DBObject, e error) {
	in := mysql.In(ids)
	sql := "select a.uid,gender,nickname,avatar,province,city,height,aboutme,avatarlevel,tag,stat,star,phonestat, certify_video,homeprovince,homecity,workarea,workunit,job,school,trade,edu,birthday,requireprovince,requirecity,minage,maxage,minheight,maxheight,minedu,needtag,hardrequire,b.workplaceid,IFNULL(building.lat,0),IFNULL(building.lng,0),IFNULL(building.`address`,''),dynamic_img from user_main a LEFT JOIN user_detail b on a.uid=b.uid LEFT JOIN building on building.placeid=b.workplaceid where a.uid" + in
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	obj := make(map[interface{}]cachedb.DBObject)
	for rows.Next() {
		var user UserViewItem
		var birthday string
		var mtag, tagstr string
		user.Ltag = NewLocaltag()
		if e = rows.Scan(&user.Uid, &user.Gender, &user.Nickname, &user.Avatar, &user.Province, &user.City, &user.Height, &user.Aboutme, &user.Avatarlevel, &mtag, &user.Stat, &user.Star, &user.CertifyPhone, &user.CertifyVideo, &user.Homeprovince, &user.Homecity, &user.WorkPlaceName, &user.Workunit, &user.Job, &user.School, &user.Trade, &user.Edu, &birthday, &user.Require.Province, &user.Require.City, &user.Require.Minage, &user.Require.Maxage, &user.Require.Minheight, &user.Require.Maxheight, &user.Require.Minedu, &tagstr, &user.Require.Hardrequire, &user.WorkPlaceId, &user.WorkLat, &user.WorkLng, &user.WorkPlaceAddress, &user.Dynamic_img); e != nil {
			return nil, e
		}
		u.Tag = strings.Split(mtag, ",")
		user.Needtag = strings.Split(tagstr, ",")
		user.Birthday, _ = utils.ToTime(birthday, format.TIME_LAYOUT_1)
		user.Age = utils.BirthdayToAge(user.Birthday)
		obj[user.Uid] = &user
	}
	sql = "select uid,content,gender,tm,timeout,`range`,min_age,max_age,min_height,max_height,star,income,certify_phone,certify_video,certify_idcard from user_tag where timeout > ? and uid" + in
	lrows, e := mdb.Query(sql, utils.Now)
	if e != nil {
		return nil, e
	}
	defer lrows.Close()
	var lt Localtag = NewLocaltag()
	var tm, timeout string
	var uid uint32

	for lrows.Next() {
		if e = lrows.Scan(&uid, &lt.Content, &lt.Req.Gender, &tm, &timeout, &lt.Req.Range, &lt.Req.MinAge, &lt.Req.MaxAge, &lt.Req.MinHeight, &lt.Req.MaxHeight, &lt.Req.Star, &lt.Req.Income, &lt.Req.CertifyPhone, &lt.Req.CertifyVideo, &lt.Req.CertifyIDcard); e != nil {
			return nil, e
		}
		lt.Tm, _ = utils.ToTime(tm, format.TIME_LAYOUT_1)
		lt.Timeout, _ = utils.ToTime(timeout, format.TIME_LAYOUT_1)
		lt.HasReq = !lt.Req.NoRequirement()

		user, ok := obj[uid]
		if ok {
			user.(*UserViewItem).Ltag = lt
		}
	}
	return obj, nil
}

func (u *UserViewItem) Expire() int {
	return 400
}

func (u *UserViewItem) GetAge() int {
	return u.Age
}

//MatchMyLocaltag检查对方是否符合自己的本地标签填写的条件
//
//	distence:距离（公里）
func (u *UserViewItem) MatchMyLocaltag(target *UserViewItem, distence float64) bool {
	return false
	// if u.Ltag.Timeout.Before(utils.Now) {
	// 	return false
	// }
	// if target.Uid == u.Uid {
	// 	return true
	// }
	// return u.Ltag.Req.MatchMyRequirement(target.Gender, "", "", distence, target.Age, target.Height, target.Star, target.Income, 0, target.CertifyPhone, target.CertifyVideo, target.CertifyIDcard)
}
