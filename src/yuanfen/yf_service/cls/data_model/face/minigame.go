package face

import (
	"errors"
	"math/rand"
	"time"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/general"
)

var gamemap map[uint32]*FaceItem = map[uint32]*FaceItem{}

//push小游戏消息
func pushMiniGame(f_uid uint32, t_uid uint32, game_id int, result int, item *FaceItem, game_name string) (msgid uint64, e error) { //id int,
	content := make(map[string]interface{})
	content["type"] = common.MSG_TYPE_MINI_GAME
	content["game_id"] = game_id
	content["game_name"] = game_name
	data := make(map[string]interface{})
	data["result"] = result
	data["pic"] = item.Pic
	data["ico"] = item.Ico
	if len(item.Res) > result {
		data["resultimg"] = item.Res[result-1]
	}
	content["data"] = data
	return general.SendMsg(f_uid, t_uid, content, "")
}

func GameSend(uid, toid uint32, game_id uint32) (result int, msgid uint64, resultimg string, e error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	var game_name string
	switch game_id { // (从1开始计数)
	case 1: //猜拳
		result = r.Intn(3) + 1
		game_name = "猜拳"
	case 2: //骰子
		result = r.Intn(6) + 1
		game_name = "骰子"
	}
	if item, ok := gamemap[game_id]; ok {
		msgid, e = pushMiniGame(uid, toid, int(game_id), result, item, game_name)
		if len(item.Res) > result {
			resultimg = item.Res[result]
		}
	} else {
		return 0, 0, "", errors.New("Invalid GameID")
	}
	return
}

// func GameDice(uid, toid uint32) (result int, msgid uint64, e error) {
// 	r := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	result = r.Intn(6)
// 	msgid, e = pushMiniGame(uid, toid, 2, result)
// 	return
// }
