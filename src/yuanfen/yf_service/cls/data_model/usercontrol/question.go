package usercontrol

import (
	"errors"
	// "yf_pkg/utils"
	"yuanfen/yf_service/cls/common"
	"yuanfen/yf_service/cls/data_model/user_overview"
)

type Question struct {
	Id       uint32 `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

var qMapMen []*Question
var qMapWomen []*Question

var qMenCount, qWomenCount int

func InitQuestion() (e error) {
	rows, err := mdb.Query("select id,question,gender from question_config where active=1")
	if err != nil {
		return err
	}
	defer rows.Close()
	qMapMen = make([]*Question, 0, 0)
	qMapWomen = make([]*Question, 0, 0)
	for rows.Next() {
		var id uint32
		var question string
		var gender int
		if err := rows.Scan(&id, &question, &gender); err != nil {
			return err
		}
		q := &Question{id, question, ""}
		switch gender {
		case 1:
			qMapMen = append(qMapMen, q)
			qMenCount++
		case 2:
			qMapWomen = append(qMapWomen, q)
			qWomenCount++
		}
	}
	return
}

func GetUidQuestion(uid uint32) (qs []interface{}, e error) {
	ue, err2 := user_overview.GetUserObjects(uid)
	if err2 != nil {
		return nil, err2
	}
	mp, ok := ue[uid]
	if !ok {
		return nil, errors.New("读取错误")
	}
	rows, err := mdb.Query("select qid,answer from user_question where uid=?", uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	umap := make(map[uint32]string)
	for rows.Next() {
		var qid uint32
		var answer string
		if err := rows.Scan(&qid, &answer); err != nil {
			return nil, err
		}
		umap[qid] = answer
	}
	qs = make([]interface{}, 0, 0)
	switch mp.Gender {
	case common.GENDER_MAN:
		for _, v := range qMapMen {
			q := &Question{v.Id, v.Question, ""}
			if answer, ok := umap[v.Id]; ok {
				q.Answer = answer
			}
			qs = append(qs, q)
		}
	case common.GENDER_WOMAN:
		for _, v := range qMapWomen {
			q := &Question{v.Id, v.Question, ""}
			if answer, ok := umap[v.Id]; ok {
				q.Answer = answer
			}
			qs = append(qs, q)
		}
	}
	return qs, nil
}

func SetQuestion(uid uint32, qid uint32, answer string) (e error) {
	if answer == "" { //删除回答
		_, err := mdb.Exec("delete from user_question where uid=? and qid=?", uid, qid)
		if err != nil {
			return err
		}
		var count int
		err = mdb.QueryRowFromMain("select count(*) from user_question where uid=?", uid).Scan(&count)
		if err != nil {
			return err
		}
		_, err = mdb.Exec("update user_detail set answercount=? where uid=?", count, uid)
		if err != nil {
			return err
		}
	} else {
		var oldcount int
		err := mdb.QueryRow("select count(*) from user_question where uid=?", uid).Scan(&oldcount)
		if err != nil {
			return err
		}
		_, err = mdb.Exec("replace into user_question (uid,qid,answer)values(?,?,?)", uid, qid, answer)
		if err != nil {
			return err
		}

		if oldcount == 0 {
			_, err = mdb.Exec("update user_detail set answercount=? where uid=?", 1, uid)
			if err != nil {
				return err
			}
			NotifyAndDelInviteList(uid, INVITE_KEY_REQUIRE)
		} else {
			var count int
			err = mdb.QueryRowFromMain("select count(*) from user_question where uid=?", uid).Scan(&count)
			if err != nil {
				return err
			}
			_, err = mdb.Exec("update user_detail set answercount=? where uid=?", count, uid)
			if err != nil {
				return err
			}
		}
	}
	return
}

func GetQuestionCount(gender int) int {
	switch gender {
	case common.GENDER_MAN:
		return qMenCount
	case common.GENDER_WOMAN:
		return qWomenCount
	}
	return qMenCount
}
