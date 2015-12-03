package user_overview

import (
	"errors"
	"yf_pkg/cachedb"
	"yf_pkg/mysql"
	"yf_pkg/utils"
)

type PhotoItem struct {
	AlbumId uint32 `json:"albumid"` //图片ID
	Pic     string `json:"pic"`     //图片URL
}
type UserPhoto struct {
	Uid       uint32       `json:"uid"`       //uid
	PhotoList []*PhotoItem `json:"photolist"` //形象照列表
}

//单个用户获取用户形象照
func GetUserPhoto(uid uint32) (result *UserPhoto, e error) {
	result = &UserPhoto{}
	e = cachedb2.Get(uid, result)
	return
}

//批量获取用户形象照
func GetUserPhotos(uidlist ...uint32) (obj map[uint32]*UserPhoto, e error) {
	if len(uidlist) == 0 {
		return
	}
	users := make(map[interface{}]cachedb.DBObject)
	for _, v := range uidlist {
		user := new(UserPhoto)
		user.Uid = v
		user.PhotoList = make([]*PhotoItem, 0, 0)
		users[utils.Uint32ToString(v)] = user
	}
	e = cachedb2.GetMap(users, NewUserPhoto)
	obj1 := make(map[uint32]*UserPhoto)
	if e != nil {
		return nil, e
	} else {
		for id, user := range users {
			uid, e := utils.ToUint32(id)
			if e != nil {
				return nil, e
			}
			// fmt.Println(fmt.Printf("GetUserProtectsr %v ,%v", id, user))
			if user != nil {
				obj1[uid] = user.(*UserPhoto)
			}
		}
	}
	return obj1, nil
}

//清除用户形象照缓存
func ClearUserPhoto(uid uint32) (e error) {
	return cachedb2.ClearCache(NewUserPhoto(uid))
}

func NewUserPhoto(uid interface{}) cachedb.DBObject {
	user := &UserPhoto{}
	switch v := uid.(type) {
	case uint32:
		user.Uid = v
	}
	return user
}

func (u *UserPhoto) ID() (id interface{}, ok bool) {
	return u.Uid, u.Uid != 0
}

func (u *UserPhoto) Save(mysqldb *mysql.MysqlDB) (id interface{}, e error) {
	return nil, errors.New("not implemented")
}

func (u *UserPhoto) Get(id interface{}, mysqldb *mysql.MysqlDB) (e error) {
	switch v := id.(type) {
	case uint32:
		u.Uid = v
	}

	rows, err := mdb.Query("select albumid,pic,user_detail.avatar from user_photo_album left join user_detail on user_detail.uid=user_photo_album.uid where uid=? order by create_time LIMIT 20", id) //req.Uid
	if err != nil {
		return err
	}
	defer rows.Close()
	list := make([]*PhotoItem, 1, 20)
	if rows.Next() {
		photo := new(PhotoItem)
		var avatar string
		if err := rows.Scan(&photo.AlbumId, &photo.Pic, &avatar); err != nil {
			return err
		}
		if avatar == photo.Pic {
			list[0] = photo
		} else {
			list = append(list, photo)
		}
	}
	if list[0] == nil {
		list = list[1:]
	}
	u.PhotoList = list
	return nil
}

func (u *UserPhoto) GetMap(ids []interface{}, mysqldb *mysql.MysqlDB) (objs map[interface{}]cachedb.DBObject, e error) {
	in := mysql.In(ids)
	sql := "select user_photo_album.uid,albumid,pic,user_detail.avatar from user_photo_album left join user_detail on user_detail.uid=user_photo_album.uid where user_photo_album.uid " + in + " order by user_photo_album.uid"
	rows, e := mdb.Query(sql)
	if e != nil {
		return nil, e
	}
	defer rows.Close()

	obj := make(map[interface{}]cachedb.DBObject)
	for rows.Next() {
		photo := new(PhotoItem)
		var avatar string
		var uid uint32
		if e = rows.Scan(&uid, &photo.AlbumId, &photo.Pic, &avatar); e != nil {
			return nil, e
		}

		if v, ok := obj[uid]; ok {
			list := v.(*UserPhoto).PhotoList
			if avatar == photo.Pic {
				list[0] = photo
			} else {
				list = append(list, photo)
			}
			v.(*UserPhoto).PhotoList = list
		} else {
			user := new(UserPhoto)
			user.Uid = uid
			list := make([]*PhotoItem, 1, 20)
			if avatar == photo.Pic {
				list[0] = photo
			} else {
				list = append(list, photo)
			}
			user.PhotoList = list
			obj[uid] = user
		}
	}
	for _, v := range obj {
		if v.(*UserPhoto).PhotoList[0] == nil {
			v.(*UserPhoto).PhotoList = v.(*UserPhoto).PhotoList[1:]
		}
	}
	return obj, nil
}

func (u *UserPhoto) Expire() int {
	return 600
}
