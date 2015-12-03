package dynamics

import (
	"errors"
	"yf_pkg/service"
	"yuanfen/redis_db"
	"yuanfen/yf_service/cls/data_model/dynamics"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

/*
编辑后台添加优秀动态到动态推荐库

URL: /s/user/IAddGoodDynamic

参数：
	id:[uint32]推荐动态id
*/
func (dm *DynamicsModule) SecIAddGoodDynamic(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var id uint32
	if e = req.Parse("id", &id); e != nil {
		return
	}
	dy, e := dynamics.GetDynamicById(id)
	if e != nil {
		return
	}
	if dy.Status != dynamics.DYNAMIC_STATUS_OK {
		e = errors.New("该动态状态不正常")
		return
	}
	u, e := user_overview.GetUserObject(dy.Uid)
	if e != nil {
		return
	}
	// 添加到新省动态的集合中
	e = dm.rdb.ZAdd(redis_db.REDIS_DYNAMIC, dynamics.MakeExProvinceDyanmicKey(u.Province), dynamics.MakeDynamicScore(id, u.Gender, 16), dynamics.MakeDynamicKey(id, u.Uid))
	return
}
