package usercontrol

type SchoolItem struct {
	School string `json:"school"`
	Owner  string `json:"owner"`
	Area   string `json:"area"`
	Level  string `json:"level"`
	Edu    int    `json:"edu"`
	Tip    string `json:"tip"`
}

var schoolmap map[string][]SchoolItem

func InitSchool() (e error) {
	schoolmap = make(map[string][]SchoolItem)
	rows, err := mdb.Query("select school,owner,area,level,tip,edu from school")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var item SchoolItem
		if err := rows.Scan(&item.School, &item.Owner, &item.Area, &item.Level, &item.Tip, &item.Edu); err != nil {
			return err
		}

		for i, s := range item.School {
			ci := item.School[0:i] + string(s)
			// fmt.Println(ci + "  " + item.School)
			if list, ok := schoolmap[ci]; ok {
				list = append(list, item)
				schoolmap[ci] = list
			} else {
				list = make([]SchoolItem, 0, 0)
				list = append(list, item)
				schoolmap[ci] = list
			}
		}

	}
	return
}

//根据名称获取大学信息
func GetSchool(name string) (school SchoolItem, found bool) {
	if list, ok := schoolmap[name]; !ok {
		return school, false
	} else {
		for _, v := range list {
			if v.School == name {
				return v, true
			}
		}
	}
	return school, false
}

func SearchSchool(input string, cur, ps int) (schools []SchoolItem, total int, e error) {
	schools = make([]SchoolItem, 0, 0)
	if cur < 1 {
		cur = 1
	}
	begin := 0
	if cur > 1 {
		begin = (cur - 1) * ps
	}
	end := cur * ps
	// fmt.Println(fmt.Sprintf("input %v begin %v end %v", input, begin, end))
	if list, ok := schoolmap[input]; ok {
		// schools = list[begin:end]
		for i, item := range list {
			if (i >= begin) && (i < end) {
				schools = append(schools, item)
				total++
			}
		}
	}
	return
}
