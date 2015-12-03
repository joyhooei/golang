package general

import "yf_pkg/utils"

/*
版本升级对象
*/
type Version struct {
	Ver     int    `json:"ver"`      // 下载版本
	Title   string `json:"title"`    // 显示版本
	IsForce int    `json:"is_force"` // 是否强制升级
	Summary string `json:"summary"`  // 升级描述
	Url     string `json:"url"`      // 下载地址
	Size    string `json:"size"`     // 包下载地址
}

// c_uid,c_sid 渠道， ver 版本号 sts_type 0 android 1 ios
func CheckUpdate(c_uid, c_sid, ver string, sys_type int) (v Version, e error) {
	if c_uid == "" || c_sid == "" || ver == "" {
		return
	}
	// 从缓存获取
	if exists, version, e := readVersionCache(c_uid, c_sid, ver); exists && e == nil {
		return version, nil
	}
	s := "select av.ver,avc.title,avc.is_force,avc.summary,avc.size from app_version as av left join app_version_config as avc on av.ver = avc.ver  where (av.c_uid = '0' or av.c_uid = ?) and (av.c_sid = '0' or av.c_sid = ?) and av.ver > ? and av.status = 1 and avc.sys_type = ? order by ver desc limit 1 "
	rows, e := mdb.Query(s, c_uid, c_sid, ver, sys_type)
	if e != nil {
		mlog.AppendObj(e, "CheckUpdate is error")
		return
	}
	defer rows.Close()
	if rows.Next() {
		if e = rows.Scan(&v.Ver, &v.Title, &v.IsForce, &v.Summary, &v.Size); e != nil {
			mlog.AppendObj(e, "CheckUpdate scan result is error")
			return
		}
	}
	// 如果最新版本为非强制升级，则需要检测之间版本是否有强制升级，如果有则需要将该版本修改为强制升级
	if v.IsForce == 0 {
		if old_ver, e := utils.ToInt(ver); e == nil && v.Ver-old_ver > 1 {
			s2 := "select count(*) from app_version as av left join app_version_config as avc on av.ver = avc.ver  where (av.c_uid = ? or av.c_uid = '0') and (av.c_sid = ? or av.c_sid = '0')and av.ver > ?  and av.ver<? and av.status = 1  and avc.`is_force`=1 and avc.sys_type = ?"
			var cnt int
			if e2 := mdb.QueryRow(s2, c_uid, c_sid, old_ver, v.Ver, sys_type).Scan(&cnt); e2 != nil {
				return v, e2
			}
			if cnt > 0 {
				v.IsForce = 1
			}
		}
	}
	// 写入缓存
	if e = writeVersionCache(c_uid, c_sid, ver, v); e != nil {
		mlog.AppendObj(e, "CheckUpdate set cache is error")
	}
	return
}
