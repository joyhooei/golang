package certify

import (
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/notify"
)

var no_can string = "no_can"
var no_bal string = "no_bal"
var cmd string = "cmd"
var tip_msg string = "tip"
var no_avatar string = "no_avatar"

// 填充弹框提示配置
func fillTips() {
	for key, _ := range pri_key {
		tip := make(map[string]string)
		tip[cmd] = notify.CMD_HONESTY
		tip[no_bal] = ""
		tip[tip_msg] = ""
		tip[no_avatar] = "没有真实头像很难得到对方的回应哦，立即上传自己的真实头像吧"
		switch key {
		case common.PRI_SAYHI:
			tip[no_can] = "个人资料不完整，将很难得到对方的回应哦~请至少提升诚信度到2星级"
			tip[no_bal] = "您当前的诚信星级每天只能打招呼%v次，诚信越高，打招呼次数将越多"
			tip[tip_msg] = "去提升星级"
		case common.PRI_CHAT:
			tip[no_can] = "个人资料不完整，将很难得到对方的回应哦~请至少提升诚信度到2星级"
			tip[no_bal] = "您当前的诚信星级每天只能私聊或打招呼%v次，诚信越高，聊天人数将越多"
			tip[tip_msg] = "去提升星级"
		case common.PRI_PURSUE:
			tip[no_can] = "追求她需要让她看到你的完整资料哦~请至少提升诚信度到3星级"
			tip[no_bal] = "您当前的诚信星级每天只能发起追求%v次，诚信越高，发起次数将越多"
			tip[tip_msg] = "去提升星级"
		case common.PRI_FOLLOW:
			tip[no_can] = "0星诚信度不能关注别人哦~没有诚信度将很难在慕慕中交到朋友"
			tip[no_bal] = "您当前的诚信星级最多关注%v个人，诚信越高，关注人数越多"
			tip[tip_msg] = "去提升星级"
		case common.PRI_NEARMSG:
			tip[no_can] = "附近的人需要看到你的样子，请先去上传一张自己的真实头像"
			tip[tip_msg] = "去提升星级"
			tip[no_avatar] = "附近的人需要看到你的样子，请先去上传一张自己的真实头像"
		case common.PRI_BIGIMG:
			//	tip[no_can] = "基于公平原则，你需要先完善自己的资料才能查看别人的，请至少提升诚信度到3星级"
			tip[no_can] = "为了保证公平的原则，您需要先完善自己资料才能查看他人的照片哦，请至少提升诚信度到3星级"
			tip[tip_msg] = "去提升星级"
		case common.PRI_CONTACT:
			tip[no_can] = "需要手机认证，才能查看对方的 \"联系方式\"，还有更多认证特权点击查看"
			tip[cmd] = notify.CMD_PHONE_PRI
			tip[tip_msg] = "查看认证特权"
		case common.PRI_PRIVATE_PHOTOS:
			tip[no_can] = "需要视频认证，才能查看对方的 \"私密照\"，还有其他4大特权点击查看"
			tip[cmd] = notify.CMD_VIDEO_PRI
			tip[tip_msg] = "查看认证特权"
		case common.PRI_SEARCH:
			tip[no_can] = "需要视频认证，才能使用\"高级筛选\" 功能，还有其他4大特权点击查看"
			tip[cmd] = notify.CMD_VIDEO_PRI
			tip[tip_msg] = "查看认证特权"
		case common.PRI_NEARMSG_FILTER:
			tip[no_can] = "需要视频认证，才能使用 \"定向\" 功能，还有其他4大特权点击查看"
			tip[cmd] = notify.CMD_VIDEO_PRI
			tip[tip_msg] = "查看认证特权"
		case common.PRI_SEE_REQUIRE:
			tip[no_can] = "需要身份证认证，才能查看用户的 \"择友问答\"，还有更多特权点击查看"
			tip[cmd] = notify.CMD_IDCARD_PRI
			tip[tip_msg] = "查看认证特权"
		case common.PRI_INVITE_NOTIFY:
			tip[no_can] = "邀请发出后对方也会查看你的资料哦，请至少提升自己的诚信度到3星级"
			tip[cmd] = notify.CMD_HONESTY
			tip[tip_msg] = "去提升等级"

		case common.PRI_AWARD_PHONECARD:
			tip[no_can] = "请先完成手机认证后再领取奖品"
			tip[cmd] = notify.CMD_PHONE_PRI
			tip[tip_msg] = "去认证"

		default:
			tip[cmd] = ""

		}
		pri_tips_map[key] = tip
	}
}
