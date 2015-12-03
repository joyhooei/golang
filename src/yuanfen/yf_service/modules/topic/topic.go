package topic

import (
	"errors"
	"fmt"
	"math/rand"
	"yf_pkg/log"
	"yf_pkg/mysql"
	"yf_pkg/redis"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/yf_service/cls"
	"yuanfen/yf_service/cls/data_model/general"
	"yuanfen/yf_service/cls/data_model/topic"
	"yuanfen/yf_service/cls/message"
)

type TopicModule struct {
	log   *log.MLogger
	mdb   *mysql.MysqlDB
	rdb   *redis.RedisPool
	cache *redis.RedisPool
}

func (sm *TopicModule) Init(env *service.Env) (err error) {
	sm.log = env.Log
	sm.mdb = env.ModuleEnv.(*cls.CustomEnv).MainDB
	sm.rdb = env.ModuleEnv.(*cls.CustomEnv).MainRds
	sm.cache = env.ModuleEnv.(*cls.CustomEnv).CacheRds

	return
}

func (sm *TopicModule) SecList(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var rid int
	var dir string
	var cur, ps int
	var lat, lng float64
	var refresh bool
	if err := req.Parse("cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := req.ParseOpt("rid", &rid, topic.RANGE_DEFAULT, "direction", &dir, "=", "refresh", &refresh, false, "lat", &lat, rand.Float64()*50+5, "lng", &lng, rand.Float64()*60+70); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if rid == topic.RANGE_UNKNOWN {
		rid = topic.RANGE_DEFAULT
	}
	rg := topic.Ranges.NextLevel(rid, dir, lat, lng)
	list, pages, err := topic.Discovery(req.Uid, rg.Radius, lat, lng, cur, ps, refresh)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, err.Error())
	}
	res := make(map[string]interface{})
	topics := make(map[string]interface{})
	topics["list"] = list
	topics["pages"] = pages
	res["topics"] = topics
	res["rid"] = rg.Id
	res["name"] = rg.Name
	result["res"] = res
	return
}

func (sm *TopicModule) SecCreate(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var capacity int
	var title, tag, pics string
	if err := req.Parse("title", &title, "tag", &tag, "capacity", &capacity, "pics", &pics); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if len(title) < 4 {
		return service.NewError(service.ERR_INVALID_PARAM, "", "话题名称长度太短")
	}
	if len(tag) < 2 {
		return service.NewError(service.ERR_INVALID_PARAM, "", "话题标签长度太短")
	}
	if len(pics) < 5 {
		return service.NewError(service.ERR_INVALID_PARAM, "", "请上传至少一张照片")
	}

	tid, timeout, err := topic.CreateTopic(req.Uid, title, tag, capacity, pics)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	res["tid"] = tid
	res["timeout"] = timeout
	res["tag"] = topic.RoomId(tid)
	result["res"] = res
	message.SendMessage(message.CREATETOPIC, message.CreateTopic{req.Uid, tid}, result)
	return
}

func (sm *TopicModule) SecIClose(req *service.HttpRequest, result map[string]interface{}) (e error) {
	if !general.IsAdmin(req.Uid) {
		return errors.New("permission denied")
	}
	var tid uint32
	if err := req.ParseOpt("tid", &tid, 0); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := topic.IClose(tid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("close topic failed : %v", err.Error()))
	}
	return
}

func (sm *TopicModule) SecClose(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.ParseOpt("tid", &tid, 0); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := topic.CloseTopic(tid, req.Uid)
	if err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("close topic failed : %v", err.Error()))
	}
	return
}

func (sm *TopicModule) SecReport(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.Parse("tid", &tid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := topic.ReportTopic(tid)
	if err != nil {
		return err
	}
	return
}

func (sm *TopicModule) SecHistory(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var cur, ps int
	if err := req.ParseOpt("cur", &cur, 1, "ps", &ps, 10); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := topic.JoinHistory(req.Uid, cur, ps)
	if err != nil {
		return err
	}
	pages := utils.PageInfo(total, cur, ps)
	res := make(map[string]interface{})
	topics := make(map[string]interface{})
	topics["list"] = list
	topics["pages"] = pages
	res["topics"] = topics
	result["res"] = res
	return
}

func (sm *TopicModule) SecDetail(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.Parse("tid", &tid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	topic, err := topic.TopicDetail(tid)
	if err != nil {
		return err
	}
	result["res"] = topic
	return
}

func (sm *TopicModule) SecJoin(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.Parse("tid", &tid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	return topic.JoinTopic(req.Uid, tid, result)
}

//只是不在聊天室，但还在话题中
func (sm *TopicModule) SecOut(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.Parse("tid", &tid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := topic.LeaveTopic(req.Uid, tid, result)
	if err != nil {
		return err
	}
	return
}

func (sm *TopicModule) SecLeave(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.Parse("tid", &tid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	err := topic.LeaveTopic(req.Uid, tid, result)
	if err != nil {
		return err
	}
	return
}

func (sm *TopicModule) SecUsers(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	var cur, ps int
	if err := req.Parse("tid", &tid, "cur", &cur, "ps", &ps); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	list, total, err := topic.TopicUsers(tid, cur, ps)
	if err != nil {
		return err
	}
	pages := utils.PageInfo(int(total), cur, ps)
	res := make(map[string]interface{})
	users := make(map[string]interface{})
	users["list"] = list
	users["pages"] = pages
	res["users"] = users
	result["res"] = res
	return
}

func (sm *TopicModule) SecKick(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid, uid, forTm uint32
	if err := req.Parse("tid", &tid, "uid", &uid, "for", forTm); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	//检查是否是话题创建者
	topics, err := topic.GetTopics(tid)
	if err != nil {
		return err
	}
	t, _ := topics[tid]
	if t == nil {
		return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("cannot find topic info of %v", tid), "该话题不存在")
	}
	if t.Uid != req.Uid {
		return service.NewError(service.ERR_INVALID_USER, "only topic owner can kick users", "只有话题创建者才能踢人")
	}
	if err = topic.Kick(uid, tid, forTm); err != nil {
		return service.NewError(service.ERR_INTERNAL, fmt.Sprintf("kick user error : %v", err.Error()))
	}
	return
}

func (sm *TopicModule) SecSend(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	var content interface{}
	if err := req.Parse("tid", &tid, "msg", &content); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	switch value := content.(type) {
	case map[string]interface{}:
		msgid, offlines, err := topic.SendMsg(req.Uid, tid, value)
		if err != nil {
			return err
		}

		res := make(map[string]interface{})
		res["msgid"] = msgid
		res["offline"] = offlines
		result["res"] = res
		return
	default:
		return service.NewError(service.ERR_INVALID_PARAM, fmt.Sprintf("msg must be a json:%v", content))
	}
	return
}

func (sm *TopicModule) SecDelMessage(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	var msgid uint64
	if err := req.Parse("tid", &tid, "msgid", &msgid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	return topic.DelMessage(req.Uid, tid, msgid)
}

func (sm *TopicModule) ClearCache(req *service.HttpRequest, result map[string]interface{}) (e error) {
	var tid uint32
	if err := req.Parse("tid", &tid); err != nil {
		return service.NewError(service.ERR_INVALID_PARAM, err.Error())
	}
	if err := topic.ClearCache(tid); err != nil {
		return
	}

	res := make(map[string]interface{})
	result["res"] = res
	return
}
