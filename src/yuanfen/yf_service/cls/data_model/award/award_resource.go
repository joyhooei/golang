package award

// 获取所有奖品列表，返回奖品map
func GetAwardMap() (map[uint32]*Award, error) {
	// 先读取cache数据
	exists, awards, e := readAwardCache()
	if exists && awards != nil && len(awards) > 0 {
		return genAwardMap(awards), e
	}
	sql := "select id,name,price,type,img,info,unit,game_img,pushflag,show_type from award_config "
	rows, err := mdb.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	arr := make([]*Award, 0, 20)
	for rows.Next() {
		a := new(Award)
		e := rows.Scan(&a.Id, &a.Name, &a.Price, &a.Atype, &a.Img, &a.Info, &a.Unit, &a.Game_img, &a.PushFlag, &a.ShowType)
		if e != nil {
			return nil, e
		}
		arr = append(arr, a)
	}
	// 写入cache
	if e := writeAwardCache(arr); e != nil {
		return nil, e
	}
	return genAwardMap(arr), err
}

func GetAwardById(id uint32) (a *Award, e error) {
	m, e := GetAwardMap()
	if e != nil {
		return nil, e
	}
	if a, ok := m[id]; ok {
		return a, nil
	}
	return
}

func genAwardMap(awards []*Award) map[uint32]*Award {
	m := make(map[uint32]*Award)
	for _, a := range awards {
		m[a.Id] = a
	}
	return m
}
