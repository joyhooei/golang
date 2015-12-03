package main

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"code.serveyou.cn/common"
	"code.serveyou.cn/location"
	"code.serveyou.cn/model"
	"code.serveyou.cn/pkg/format"
)

type TransactionHandler struct {
	tdb *TransactionDBAdapter
	csr []uint //客服ID列表
}

func NewTransactionHandler(db *sql.DB) (t *TransactionHandler, err common.Error) {
	t = new(TransactionHandler)
	t.tdb = NewTransactionDBAdapter(db)
	var e error
	t.csr, e = t.tdb.GetCSR()
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetCSR error : %v", e.Error()))
	}

	go t.SendNotification()
	return
}

func (t *TransactionHandler) SendNotification() {
	notified := make(map[uint64]bool)
	for {
		fmt.Println("check")
		orders, err := t.tdb.GetNeedNotifyOrders()
		if err != nil {
			fmt.Println(err.Error())
			glog.Append(err.Error())
		} else {
			for _, order := range orders {
				if _, found := notified[order.Id]; !found {
					extras := make(map[string]interface{})
					extras["type"] = common.NOTI_ORDER
					extras["oid"] = order.Id
					alias := make([]common.UIDType, 0, 1)
					alias = append(alias, order.Customer)
					e := common.PushNotification(alias, nil, "订单完成提醒", "阿姨完成服务了吗？", extras)
					if e != nil {
						fmt.Println(e.Error())
						glog.Append(e.Error())
					} else {
						notified[order.Id] = true
					}
				}
			}
		}
		//time.Sleep(1 * time.Minute)
		time.Sleep(10 * time.Second)
	}
}
func (t *TransactionHandler) ProcessSec(cmd string, r *http.Request, uid common.UIDType, devid string, body string, result map[string]interface{}) (err common.Error) {
	switch cmd {
	case "place_order":
		err = t.placeOrder(result, r, uid, body)
	case "get_order":
		err = t.getOrder(result, r, uid, body)
	case "cancel_order":
		err = t.cancelOrder(result, r, uid, body)
	case "confirm_order":
		err = t.confirmOrder(result, r, uid, body)
	case "complete_order":
		err = t.completeOrder(result, r, uid, body)
	case "suspend_order":
		err = t.suspendOrder(result, r, uid, body)
	case "rank":
		err = t.rank(result, r, uid, body)
	case "list_orders":
		err = t.listOrders(result, r, uid, body)
	case "list_trans":
		err = t.listTransactions(result, r, uid, body)
	case "list_goods":
		err = t.listGoods(result, r, uid, body)
	case "buy_goods":
		err = t.buyGoods(result, r, uid, body)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown sec command : "+cmd)
		return
	}
	return
}

func (t *TransactionHandler) Process(cmd string, r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	switch cmd {
	case "comments":
		err = t.comments(r, body, result)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown command : "+cmd)
		return
	}
	return
}

func (t *TransactionHandler) placeOrder(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"role"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	role, e := format.ParseUint(m["role"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [role] error : %v", e.Error()))
		return
	}

	switch role {
	case common.ROLE_HKP:
		err = t.placeHKPOrder(result, uid, m)
	default:
		err = common.NewError(common.ERR_INVALID_PARAM, "unknown role : "+m["role"])
	}

	return
}

//选择类似阿姨，直接放入order中
func (t *TransactionHandler) setSimilarHKPs(elems []location.Element, noTimeHKPs map[common.UIDType]bool, order *model.HKPOrder) {
	for k, _ := range noTimeHKPs {
		fmt.Printf("\t%v\n", k)
	}

	//给有时间的阿姨打分，打分规则是：
	//	评价>3分得分+N，N的计算方法为评价-3
	//	年龄相差3岁以内的+1
	//	价格相近+2
	//	籍贯相同+N，N取决于籍贯相同的候选人数
	//	服务超过5次，拒绝率低于30%每降低10%加1分,反之减1分
	//	服务超过5次，失败率20%不加分，每降低10%加1分，反之减1分
	//选择得分最高的X个阿姨
	var maxBirthday, minBirthday time.Time = common.InitDate, time.Now()
	var maxPrice, minPrice float32 = 0, math.MaxFloat32
	birthPlace := make(map[string]uint8)

	for _, provider := range order.Providers {
		user := Users[provider.UserId]
		if user.Birthday().After(maxBirthday) {
			maxBirthday = user.Birthday()
		}
		if user.Birthday().Before(minBirthday) {
			minBirthday = user.Birthday()
		}
		if provider.Job.Price > maxPrice {
			maxPrice = provider.Job.Price
		}
		if provider.Job.Price < minPrice {
			minPrice = provider.Job.Price
		}
		birthPlace[user.Birthplace()]++
	}

	//如果用户没选小时工，则根据平均价格确定最高和最低价格
	if len(order.Providers) == 0 {
		var totalPrice float32 = 0
		var count uint = 0
		for _, elem := range elems {
			j, found := CommunityHKPs[elem.Id]
			if found {
				u, found := j[order.Job()]
				if found {
					for _, d := range u {
						totalPrice += d.Job.Price
						count++
						fmt.Printf("totalPrice=%v, count=%v\n", totalPrice, count)
					}
				}
			}
		}
		minPrice = totalPrice / float32(count) * 0.8
		maxPrice = totalPrice / float32(count) * 1.2
		fmt.Printf("avgPrice=%v, minPrice=%v, maxPrice=%v\n", totalPrice/float32(count), minPrice, maxPrice)
	}
	score := make(common.RandomScoreElems, 0, 10)
	for _, elem := range elems {
		j, found := CommunityHKPs[elem.Id]
		if found {
			u, found := j[order.Job()]
			if found {
				for uid, d := range u {
					if _, found := noTimeHKPs[uid]; !found {
						log := fmt.Sprintf("uid=%v ", uid)
						var rank int = 0
						if d.Rank.RankAll() > 3.0 {
							rank += int(d.Rank.RankAll() - 3.0)
							log += fmt.Sprintf("rank+%v ", int(d.Rank.RankAll()-3.0))
						}
						if d.Rank.Times >= 5 {
							rank += (3 - int(d.Rank.RejectTimes*10/d.Rank.Times))
							log += fmt.Sprintf("RejectTimes+%v ", (3 - int(d.Rank.RejectTimes*10/d.Rank.Times)))
							rank += (2 - int(d.Rank.FailTimes*10/d.Rank.Times))
							log += fmt.Sprintf("FailTime+%v ", (2 - int(d.Rank.FailTimes*10/d.Rank.Times)))
						}
						us := Users[uid]
						if us.Birthday().Before(maxBirthday.Add(3*time.Hour*24*365)) && us.Birthday().After(minBirthday.Add(-3*time.Hour*24*365)) {
							rank += 1
							log += "age+1 "
						}
						if d.Job.Price < maxPrice*1.2 {
							rank += 1
							log += "maxPrice+1 "
						}
						if d.Job.Price > minPrice*0.7 {
							rank += 1
							log += "minPrice+1 "
						}
						if _, found := birthPlace[us.Birthplace()]; found {
							rank += int(birthPlace[us.Birthplace()])
							log += fmt.Sprintf("birthPlace+%v ", int(birthPlace[us.Birthplace()]))
						}
						score = append(score, common.ScoreElem{uid, elem.Id, rank, rand.Int31n(math.MaxInt32)})
						log += fmt.Sprintf("total=%v", rank)
						fmt.Println(log)
					}
				}
			}
		}
	}
	if len(score) == 0 {
		return
	}
	//打乱顺序，确保相同rank的阿姨有相等的机会
	sort.Sort(score)
	//寻找评价最高的N个HKP
	sn := len(order.Providers)
	maxRanks := make(common.ScoreElems, int(common.Variables.HKPMaxCount())-sn)
	min := 0
	for _, elem := range score {
		if maxRanks[min].Rank < elem.Rank {
			maxRanks[min] = elem
			for i, r := range maxRanks {
				if r.Rank < maxRanks[min].Rank {
					min = i
				}
			}
		}
	}
	sort.Sort(maxRanks)

	//写入订单
	i := 1
	for _, elem := range maxRanks {
		if elem.Uid != 0 {
			job := CommunityHKPs[elem.Comm][order.Job()][elem.Uid].Job
			order.AddProvider(&job, uint8(i), true)
			i += 1
		}
	}

	return
}

func (t *TransactionHandler) placeHKPOrder(result map[string]interface{}, uid common.UIDType, m map[string]string) (err common.Error) {
	suc, key := format.Contains(m, []string{"job", "persons", "start", "duration", "address", "candidates", "description"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	order := model.NewHKPOrder()
	if e := order.SetCustomer(uid); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set customer error : %v", e.Error()))
		return
	}
	if e := order.SetJobStr(m["job"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set job error : %v", e.Error()))
		return
	}
	if e := order.SetPersonsStr(m["persons"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set persons error : %v", e.Error()))
		return
	}
	if e := order.SetServiceStartStr(m["start"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set start error : %v", e.Error()))
		return
	}
	if e := order.SetDurationStr(m["duration"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set duration error : %v", e.Error()))
		return
	}
	if e := order.SetAddressStr(m["address"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set address error : %v", e.Error()))
		return
	}
	if e := order.SetStartTimeTime(time.Now()); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set start time error : %v", e.Error()))
		return
	}
	if e := order.SetDescription(m["description"]); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set description error : %v", e.Error()))
		return
	}
	fmt.Printf("order.Desc : %v\n", order.Description())

	//服务时间是否符合要求
	if order.ServiceStart().Before(time.Now()) {
		err = common.NewError(common.ERR_INVALID_PARAM, "service start time must after now")
		return
	}

	if uint(order.ServiceStart().Hour()) > common.Variables.HKPServiceStop() || uint(order.ServiceStart().Hour()) < common.Variables.HKPServiceStart() {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Service time must between %v:00 and %v:00", common.Variables.HKPServiceStart(), common.Variables.HKPServiceStop()))
		return
	}
	//添加candidates
	if m["candidates"] != "" {
		cas := strings.Split(m["candidates"], ",")
		if len(cas) > 3 {
			err = common.NewError(common.ERR_INVALID_PARAM, "too many candidates")
			return
		}

		for priority, ca := range cas {
			kv := strings.Split(ca, ":")
			if len(kv) != 2 {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse candidates error : %v format invalid", ca))
				return
			}
			uid, e := format.ParseUint(kv[0])
			if e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [%v] error : %v", ca, e.Error()))
				return err
			}
			jv, e := format.ParseUint(kv[1])
			if e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse [%v] error : %v", ca, e.Error()))
				return err
			}
			job, e := t.tdb.GetHKPJob(common.UIDType(uid), order.Job(), jv)
			if e != nil {
				err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetHKPJob error : %v", e.Error()))
				return err
			}

			if e = order.AddProvider(job, uint8(priority+1), false); e != nil {
				err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("add provider error : %v", e.Error()))
				return err
			}
		}
	}

	//直接进入人工联系阶段，以后如果需要自动分发阶段再在这里加

	if e := order.SetPhase(common.HKP_PHASE_CSR); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set phase error : %v", e.Error()))
		return
	}
	if e := order.SetPhaseTime(time.Now()); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set phase time error : %v", e.Error()))
		return
	}

	//分配客服（CSR）
	idx := rand.Intn(len(t.csr))
	fmt.Printf("idx = %v\n", idx)
	fmt.Printf("csr : %v\n", t.csr)
	if e := order.SetCSR(t.csr[idx]); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set CSR time error : %v", e.Error()))
		return
	}

	//寻找推荐阿姨
	adInfo, e := t.tdb.GetUserAddress(uid, order.Address())
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetUserAddress error : %v", e.Error()))
		return
	}
	if e = order.SetCommunityUint(adInfo.Community); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set community error : %v", e.Error()))
		return
	}
	//先要找到附近两倍范围内没时间的阿姨
	elems, e := loc.Adjacent(adInfo.Community, common.Variables.AdjacentRange()*2)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("find Adjacent error : %v", e.Error()))
		return
	}
	noTimeHKPs, e := GetHKPsNoTime(t.tdb.db, elems, order.Role(), order.ServiceStart())
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetHKPsNoTime error : %v", e.Error()))
		return
	}
	//把用户选择的阿姨也要排除在外
	for _, p := range order.Providers {
		noTimeHKPs[p.UserId] = true
	}
	//找到距离符合要求的小区
	elems, e = loc.Adjacent(adInfo.Community, common.Variables.AdjacentRange())
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("find Adjacent error : %v", e.Error()))
		return
	}
	//寻找合适的阿姨
	t.setSimilarHKPs(elems, noTimeHKPs, order)

	//写入数据库
	if e = t.tdb.AddHKPOrder(order); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddHKPOrder error : %v", e.Error()))
		return
	}
	result["status"] = "ok"
	result["orderid"] = order.OrderId()
	if tmpErr := t.tdb.UpdateAddressLastUse(order.Address()); tmpErr != nil {
		fmt.Println(tmpErr.Error())
		glog.Append(tmpErr.Error())
	}
	return
}

func (t *TransactionHandler) getHKPOrderProviders(oid uint64, jid uint) (result []map[string]interface{}, err common.Error) {
	providers, e := t.tdb.GetHKPProviders(oid, jid)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetHKPProviders error : %v", e.Error()))
		return
	}
	result = make([]map[string]interface{}, 0, 3)
	var selected common.UIDType = 0
	for uid, p := range providers {
		if p.Confirm == common.HKP_PROVIDER_CONFIRM_YES {
			selected = uid
			if !p.Recommend {
				break
			}
		}
	}
	for uid, p := range providers {
		m := make(map[string]interface{})
		if uid == selected {
			m["selected"] = "true"
		}
		m["uid"] = uint64(uid)
		m["price"] = p.Job.Price
		m["jobversion"] = p.Job.Version
		m["confirm"] = p.Confirm
		m["recommend"] = p.Recommend
		m["available_time"] = p.AvailableTime.Format(format.TIME_LAYOUT_1)
		m["attitude"] = p.Attitude()
		m["speed"] = p.Speed()
		m["quality"] = p.Quality()
		m["comment"] = p.Comment()
		uinfo, found := Users[uid]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("user %v not found", uid))
			return result, err
		}
		hkp, found := HKPs[uid][jid]
		if !found {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("user %v not an hkp", uid))
			return result, err
		}
		m["idcardtype"] = uinfo.IdCardType()
		masked, e := common.MaskIDCardNum(uinfo.IdCardNum())
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("invalid id card %v", uinfo.IdCardNum()))
			return result, err
		}
		m["idcardnum"] = masked
		m["nickname"] = uinfo.NickName()
		m["firstname"] = uinfo.FirstName()
		m["lastname"] = uinfo.LastName()
		m["sex"] = uinfo.Sex()
		m["phone"] = uinfo.Cellphone()
		m["birthday"] = uinfo.Birthday().Format(format.TIME_LAYOUT_2)
		m["birthplace"] = uinfo.Birthplace()
		m["rank"] = hkp.Rank.RankAll()
		m["rank_desc"] = hkp.Rank.RankAllDesc()
		m["times"] = hkp.Rank.SuccessTimes()
		if uinfo.HealthCardTimeout().After(time.Now()) {
			m["health"] = true
		} else {
			m["health"] = false
		}
		result = append(result, m)
	}
	return
}

func (t *TransactionHandler) getOrder(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"orderid"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	oid, e := format.ParseUint64(m["orderid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse orderid error : %v", e.Error()))
		return
	}
	order, e := t.tdb.GetOrder(oid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find order %v", oid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetOrder error : %v", e.Error()))
		}
		return
	}

	result["phase"] = order.Phase
	result["role"] = order.Role
	result["job"] = order.Job
	result["persons"] = order.Persons
	result["phase_reason"] = order.PhaseReason
	result["duration"] = order.Duration
	result["phase_time"] = order.PhaseTime.Format(format.TIME_LAYOUT_1)
	result["service_start"] = order.ServiceStart.Format(format.TIME_LAYOUT_1)
	result["order_start"] = order.StartTime.Format(format.TIME_LAYOUT_1)

	switch order.Role {
	case common.ROLE_HKP:
		result["providers"], err = t.getHKPOrderProviders(oid, order.Job)
	default:
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("unknown role : %v", order.Role))
	}
	return
}

func (t *TransactionHandler) cancelOrder(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"orderid", "reason"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	oid, e := format.ParseUint64(m["orderid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse orderid error : %v", e.Error()))
		return
	}
	order, e := t.tdb.GetOrder(oid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find order %v", oid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetOrder error : %v", e.Error()))
		}
		return
	}
	//只有在完成订单前才可以取消，其它阶段都只能让订单失败
	if order.Phase != common.HKP_PHASE_AUTO_DISPATCH && order.Phase != common.HKP_PHASE_CSR && order.Phase != common.HKP_PHASE_RECOMMEND {
		err = common.NewError(common.ERR_INVALID_PHASE, fmt.Sprintf("invalid phase [%d] to cancel order.", order.Phase))
		return
	}
	order.Phase = common.HKP_PHASE_CUSTOMER_CANCEL
	order.PhaseTime = time.Now()
	order.PhaseReason = m["reason"]

	e = t.tdb.UpdateOrder(order, nil)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateOrder error : %v", e.Error()))
	}
	return
}

func (t *TransactionHandler) confirmOrder(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"orderid", "provider"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	oid, e := format.ParseUint64(m["orderid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse orderid error : %v", e.Error()))
		return
	}
	tmp, e := format.ParseUint64(m["provider"])
	pr := common.UIDType(tmp)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse provider error : %v", e.Error()))
		return
	}
	order, e := t.tdb.GetOrder(oid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find order %v", oid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetOrder error : %v", e.Error()))
		}
		return
	}
	//只有在推荐阶段才需要确认
	if order.Phase != common.HKP_PHASE_RECOMMEND {
		err = common.NewError(common.ERR_INVALID_PHASE, fmt.Sprintf("invalid phase [%d] to confirm order.", order.Phase))
		return
	}
	order.Phase = common.HKP_PHASE_ORDER_SUCCESS
	order.PhaseTime = time.Now()

	//读取实际提供服务的阿姨评价记录
	providers, e := t.tdb.GetHKPProviders(oid, order.Job)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetHKPProviders error : %v", e.Error()))
		return
	}
	provider := model.NewHKPOrderProvider()
	for _, p := range providers {
		if p.UserId == pr {
			*provider = p
			provider.Confirm = common.HKP_PROVIDER_CONFIRM_YES
			break
		}
	}
	if provider.UserId == 0 {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("not found provider [%v] in order [%v]", pr, oid))
		return
	}
	fmt.Printf("confirmed provider is %v, confirm is %v, avaiable time is %v\n", provider.UserId, provider.Confirm, provider.AvailableTime)

	if provider.Recommend == false {
		order.ServiceStart = provider.AvailableTime
	}

	tx, e := t.tdb.Begin()
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("begin transaction error : %v", e.Error()))
		return
	}
	if e = t.tdb.UpdateHKPProvider(*provider, tx); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateHKPProvider error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}
	if e = t.tdb.AddHKPRankTimes(pr, order.Job, tx); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddHKPRankTimes error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}
	e = t.tdb.UpdateOrder(order, tx)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateOrder error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}
	txerr := tx.Commit()
	if txerr != nil {
		err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
	}

	//TODO:给阿姨发订单成功的消息

	return
}

func (t *TransactionHandler) completeOrder(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"orderid"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	oid, e := strconv.ParseUint(m["orderid"], 10, 64)
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse orderid error : %v", e.Error()))
		return
	}
	order, e := t.tdb.GetOrder(oid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find order %v", oid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetOrder error : %v", e.Error()))
		}
		return
	}
	//只有在订单完成阶段才能完成服务
	if order.Phase != common.HKP_PHASE_ORDER_SUCCESS {
		err = common.NewError(common.ERR_INVALID_PHASE, fmt.Sprintf("invalid phase [%d] to complete order.", order.Phase))
		return
	}
	//必须在服务开始后才能完成
	//TODO:记得要把下面的注释去掉
	/*
		if order.ServiceStart.After(time.Now()) {
			err = common.NewError(common.ERR_INVALID_REQUEST, "complete order error : service not start yet.")
			return
		}
	*/

	order.Phase = common.HKP_PHASE_SERVICE_COMPLETE
	order.PhaseTime = time.Now()

	e = t.tdb.UpdateOrder(order, nil)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateOrder error : %v", e.Error()))
		return
	}

	//更新服务历史
	h := model.NewHistory()
	if e = h.SetCustomer(order.Customer); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set customer error : %v", e.Error()))
		return
	}
	if e = h.SetRole(order.Role); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set role error : %v", e.Error()))
		return
	}
	if e = h.SetJob(order.Job); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set job error : %v", e.Error()))
		return
	}
	h.LastTime = order.ServiceStart
	providers, e := t.tdb.GetConfirmedHKPProviders(oid, order.Job)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetConfirmedHKPProviders error : %v", e.Error()))
		return
	}
	for uid, _ := range providers {
		if e = h.SetServiceProvider(uid); e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Set service provider error : %v", e.Error()))
			return
		}
		if e = t.tdb.AddHistory(h); e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddHistory error : %v", e.Error()))
			return
		}
	}
	return
}

func (t *TransactionHandler) suspendOrder(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"orderid", "reason"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	oid, e := format.ParseUint64(m["orderid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse orderid error : %v", e.Error()))
		return
	}
	order, e := t.tdb.GetOrder(oid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find order %v", oid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetOrder error : %v", e.Error()))
		}
		return
	}
	//只有在订单完成阶段才能中止服务
	if order.Phase != common.HKP_PHASE_ORDER_SUCCESS {
		err = common.NewError(common.ERR_INVALID_PHASE, fmt.Sprintf("invalid phase [%d] to suspend order.", order.Phase))
		return
	}

	order.Phase = common.HKP_PHASE_SERVICE_FAIL
	order.PhaseTime = time.Now()
	order.PhaseReason = m["reason"]

	if e = t.tdb.UpdateOrder(order, nil); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateOrder error : %v", e.Error()))
		return
	}

	//增加服务提供者的订单失败次数
	providers, e := t.tdb.GetConfirmedHKPProviders(oid, order.Job)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetConfirmedHKPProviders error : %v", e.Error()))
		return
	}
	fmt.Printf("Providers : %v\n", providers)
	e = t.tdb.AddHKPFailTimes(providers, order.Job, 1)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddHKPFailTimes error : %v", e.Error()))
		return
	}

	//TODO:给阿姨发订单失败的短信

	return
}

func (t *TransactionHandler) rank(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	values, e := format.ParseKVGroup(body, "\r\n", "\r\n\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	if len(values) == 0 {
		err = common.NewError(common.ERR_INVALID_FORMAT, fmt.Sprintf("no data : %v", body))
		return
	}

	suc, key := format.Contains(values[0], []string{"orderid"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	oid, e := format.ParseUint64(values[0]["orderid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse orderid error : %v", e.Error()))
		return
	}
	order, e := t.tdb.GetOrder(oid)
	if e != nil {
		if e == sql.ErrNoRows {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("cannot find order %v", oid))
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetOrder error : %v", e.Error()))
		}
		return
	}
	//只有服务完成后才能评价
	if order.Phase != common.HKP_PHASE_SERVICE_COMPLETE {
		err = common.NewError(common.ERR_INVALID_PHASE, fmt.Sprintf("invalid phase [%d] to rank.", order.Phase))
		return
	}

	//读取实际提供服务的阿姨评价记录
	providers, e := t.tdb.GetConfirmedHKPProviders(oid, order.Job)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetConfirmedHKPProviders error : %v", e.Error()))
		return
	}
	uids := []common.UIDType{}
	for _, p := range providers {
		uids = append(uids, p.UserId)
		fmt.Printf("provider : uid=%v, spd=%v, oid=%v\n", p.UserId, p.Speed(), p.OrderId)
	}
	ranks, e := t.tdb.GetHKPRanks(uids, order.Job)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetHKPRanks error : %v", e.Error()))
		return
	}

	providers_update := make([]model.HKPOrderProvider, 0)
	ranks_update := make([]model.HKPRank, 0)
	for i := 1; i < len(values); i++ {
		suc, key := format.Contains(values[i], []string{"uid", "speed", "quality", "attitude", "comment"})
		if !suc {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%s] provided in rank %d.", key, i))
			return
		}
		var tmpuid uint64
		tmpuid, e = format.ParseUint64(values[i]["uid"])
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse uid error : %v", e.Error()))
			return
		}
		uid := common.UIDType(tmpuid)
		var speed, quality, attitude uint8
		speed, e = format.ParseUint8(values[i]["speed"])
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse speed error : %v", e.Error()))
			return
		}
		if speed == 0 {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("speed must > 0"))
			return
		}
		quality, e = format.ParseUint8(values[i]["quality"])
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse quality error : %v", e.Error()))
			return
		}
		if quality == 0 {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("quality must > 0"))
			return
		}
		attitude, e = format.ParseUint8(values[i]["attitude"])
		if e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse attitude error : %v", e.Error()))
			return
		}
		if attitude == 0 {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("attitude must > 0"))
			return
		}
		provider, found := providers[uid]
		if !found {
			err = common.NewError(common.ERR_INVALID_REQUEST, fmt.Sprintf("user [uid=%v] not participate this order.", uid))
			return
		}

		if e = provider.SetRank(speed, quality, attitude); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("set rank error : %v", e.Error()))
			return
		}
		if e = provider.SetComment(values[i]["comment"]); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("set comment error : %v", e.Error()))
			return
		}
		providers_update = append(providers_update, provider)
		rank, found := ranks[uid]
		if e = rank.AddRank(uint(speed), uint(quality), uint(attitude)); e != nil {
			err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("add rank error : %v", e.Error()))
			return
		}
		ranks_update = append(ranks_update, rank)
	}

	//更新评价
	tx, e := t.tdb.Begin()
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("begin transaction error : %v", e.Error()))
		return
	}
	if e = t.tdb.UpdateHKPProviders(providers_update, tx); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateHKPProviders error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}
	if e = t.tdb.UpdateHKPRanks(ranks_update, tx); e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateHKPRanks error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}

	order.Phase = common.HKP_PHASE_RANK_COMPLETE
	order.PhaseTime = time.Now()

	e = t.tdb.UpdateOrder(order, tx)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("UpdateOrder error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}

	isFirstOrder, e := t.tdb.IsFirstOrder(uid)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("IsFirstOrder error : %v", e.Error()))
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}
	if isFirstOrder {
		_, _, _, e = common.AddBSC(common.TRANS_FIRST_ORDER, uid, order.Id, 0, float32(common.Variables.CFirstOrderBonus()), 0, "", tx)
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddBSC error : %s", e.Error()))
			tx.Rollback()
			return
		}
	}
	_, _, _, e = common.AddBSC(common.TRANS_EACH_ORDER, uid, order.Id, int(common.Variables.COrderBonus()), 0, 0, "", tx)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddBSC error : %s", e.Error()))
		tx.Rollback()
		return
	}
	txerr := tx.Commit()
	if txerr != nil {
		err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
	}
	//Push消息
	if isFirstOrder {
		extras := make(map[string]interface{})
		extras["type"] = common.NOTI_REWARD
		extras["oid"] = order.Id
		alias := make([]common.UIDType, 0, 1)
		alias = append(alias, uid)
		e = common.PushNotification(alias, nil, "首次完成订单奖励", fmt.Sprintf("恭喜您完成首次订单，奖励您%v元现金！", common.Variables.CFirstOrderBonus()), extras)
		if e != nil {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Push Notification error : %s", e.Error()))
			glog.Append(err.Error())
		}
	}

	return
}

func (t *TransactionHandler) listOrders(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"pn", "rn", "orderby", "desc"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	phases := make([]uint8, 0, 4)
	if _, found := m["phases"]; found {
		ps := strings.Split(m["phases"], ",")
		for _, p := range ps {
			phase, e := format.ParseUint8(p)
			if e != nil {
				err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse phase error : %v", e.Error()))
				return err
			}
			phases = append(phases, phase)
		}
	}
	pn, e := format.ParseUint(m["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse pn error : %v", e.Error()))
		return
	}
	rn, e := format.ParseUint(m["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse rn error : %v", e.Error()))
		return
	}
	desc := false
	if m["desc"] == "true" {
		desc = true
	}

	orders, e := t.tdb.ListOrders(uid, phases, m["orderby"], desc, pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("ListOrder error : %v", e.Error()))
		return
	}

	os := make([]interface{}, 0, 5)
	for _, order := range orders {
		o := make(map[string]interface{})
		o["oid"] = order.Id
		o["role"] = order.Role
		o["job"] = order.Job
		o["phase"] = order.Phase
		o["duration"] = order.Duration
		o["order_start"] = order.StartTime.Format(format.TIME_LAYOUT_1)
		o["service_start"] = order.ServiceStart.Format(format.TIME_LAYOUT_1)
		os = append(os, o)
	}
	result["orders"] = os
	return
}
func (t *TransactionHandler) hKPComments(uid common.UIDType, pn uint, rn uint, result map[string]interface{}) (err common.Error) {
	comments, e := t.tdb.HKPComments(uid, pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("Query database error : %v", e.Error()))
		return
	}
	cs := make([]interface{}, 0, rn)
	for _, provider := range comments {
		c := make(map[string]interface{})
		c["uid"] = provider.UserId //临时用该属性装载customerID
		c["speed"] = provider.Speed()
		c["quality"] = provider.Quality()
		c["attitude"] = provider.Attitude()
		c["comment"] = provider.Comment()
		cs = append(cs, c)
	}
	result["comments"] = cs
	return
}
func (t *TransactionHandler) comments(r *http.Request, body string, result map[string]interface{}) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"uid", "role", "pn", "rn"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}

	var role uint
	if e = common.SetRoleStr(m["role"], &role); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set role error : %v", e.Error()))
		return
	}
	var uid common.UIDType
	if e = common.SetUidStr(m["uid"], &uid); e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Set uid error : %v", e.Error()))
		return
	}

	pn, e := format.ParseUint(m["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse pn error : %v", e.Error()))
		return
	}
	rn, e := format.ParseUint(m["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse rn error : %v", e.Error()))
		return
	}

	switch role {
	case common.ROLE_HKP:
		err = t.hKPComments(uid, pn, rn, result)
	}
	return
}

func (t *TransactionHandler) listTransactions(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"pn", "rn"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	pn, e := format.ParseUint(m["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse pn error : %v", e.Error()))
		return
	}
	rn, e := format.ParseUint(m["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse rn error : %v", e.Error()))
		return
	}
	trans, e := t.tdb.ListTransactions(uid, pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("ListTransactions error : %v", e.Error()))
		return
	}

	os := make([]interface{}, 0, 5)
	for _, t := range trans {
		o := make(map[string]interface{})
		o["id"] = t.Id
		o["time"] = t.Time.Format(format.TIME_LAYOUT_1)
		o["type"] = t.Type
		o["title"] = t.Title()
		o["oid"] = t.OrderId
		o["add_score"] = t.AddScore
		o["add_balance"] = t.AddBalance
		o["add_coupons"] = t.AddCoupons
		o["desc"] = t.Desc
		o["old_score"] = t.Score
		o["old_balance"] = t.Balance
		o["old_coupons"] = t.Coupons
		os = append(os, o)
	}
	result["transactions"] = os
	return
}
func (t *TransactionHandler) listGoods(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"pn", "rn"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	pn, e := format.ParseUint(m["pn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse pn error : %v", e.Error()))
		return
	}
	rn, e := format.ParseUint(m["rn"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse rn error : %v", e.Error()))
		return
	}
	goods, e := t.tdb.ListGoods(pn, rn)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("ListGoods error : %v", e.Error()))
		return
	}

	gs := make([]interface{}, 0, 5)
	for _, g := range goods {
		gd := make(map[string]interface{})
		gd["id"] = g.Id
		gd["name"] = g.Name
		gd["pic"] = g.Pic
		gd["price"] = g.Price
		gd["detail"] = g.Detail
		gs = append(gs, gd)
	}
	result["goods"] = gs
	return
}
func (t *TransactionHandler) buyGoods(result map[string]interface{}, r *http.Request, uid common.UIDType, body string) (err common.Error) {
	m, e := format.ParseKV(body, "\r\n")
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("parse post data error : %v", e.Error()))
		return
	}
	suc, key := format.Contains(m, []string{"gid"})
	if !suc {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("no [%v] provided", key))
		return
	}
	gid, e := format.ParseUint(m["gid"])
	if e != nil {
		err = common.NewError(common.ERR_INVALID_PARAM, fmt.Sprintf("Parse gid error : %v", e.Error()))
		return
	}
	goods, e := t.tdb.GetGoods(gid)
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("GetGoods error : %v", e.Error()))
		return
	}
	tx, e := t.tdb.db.Begin()
	if e != nil {
		err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("begin transaction error : %v", e.Error()))
		return
	}
	_, _, _, e = common.AddBSC(common.TRANS_BUY_GOODS, uid, 0, 0, -(goods.Price), 0, "处理中", tx)
	if e != nil {
		if strings.Index(e.Error(), "not enough") >= 0 {
			err = common.NewError(common.ERR_NOT_ENOUGH_MONEY, e.Error())
		} else {
			err = common.NewError(common.ERR_INTERNAL, fmt.Sprintf("AddBSC error : %v", e.Error()))
		}
		txerr := tx.Rollback()
		if txerr != nil {
			err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
		}
		return
	}
	txerr := tx.Commit()
	if txerr != nil {
		err = common.NewError(common.ERR_INTERNAL, err.Error()+"; "+txerr.Error())
	}

	result["explain"] = goods.Explain
	return
}
