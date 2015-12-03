package model

import (
	"time"

	"code.serveyou.cn/common"
)

type HKPJob struct {
	Uid       common.UIDType
	JobId     uint
	Price     float32
	BeginTime time.Time
	Desc      string
	Version   uint
}

type HKPRank struct {
	Uid           common.UIDType
	JobId         uint
	Times         uint
	TotalSpeed    uint
	TotalQuality  uint
	TotalAttitude uint
	RejectTimes   uint
	FailTimes     uint
}

func NewHKPJob() (j *HKPJob) {
	return &HKPJob{0, 0, 0, common.InitDate, "", 0}
}

func NewHKPRank() (r *HKPRank) {
	return &HKPRank{0, 0, 0, 0, 0, 0, 0, 0}
}

func (r *HKPRank) AddRank(s uint, q uint, a uint) (err error) {
	r.TotalSpeed += s
	r.TotalQuality += q
	r.TotalAttitude += a
	return
}

func (r *HKPRank) SuccessTimes() (t uint) {
	return r.Times - r.FailTimes - r.RejectTimes
}
func (r *HKPRank) Quality() (v float32) {
	if r.Times == 0 {
		v = 0
	} else {
		v = float32(r.TotalQuality) / float32(r.Times-r.FailTimes-r.RejectTimes)
		if v > 5 {
			v = 5
		}
	}
	return
}
func (r *HKPRank) QualityDesc() string {
	if r.Quality() >= 4 {
		return "质量高"
	}
	return ""
}
func (r *HKPRank) Attitude() (v float32) {
	if r.Times == 0 {
		v = 0
	} else {
		v = float32(r.TotalAttitude) / float32(r.Times-r.FailTimes-r.RejectTimes)
		if v > 5 {
			v = 5
		}
	}
	return
}
func (r *HKPRank) AttitudeDesc() string {
	if r.Attitude() >= 4 {
		return "热情"
	}
	return ""
}
func (r *HKPRank) Speed() (v float32) {
	if r.Times == 0 {
		v = 0
	} else {
		v = float32(r.TotalSpeed) / float32(r.Times-r.FailTimes-r.RejectTimes)
		if v > 5 {
			v = 5
		}
	}
	return
}
func (r *HKPRank) SpeedDesc() string {
	if r.Speed() >= 4 {
		return "干活快"
	}
	return ""
}
func (r *HKPRank) RankAll() (v float32) {
	if r.Times == 0 {
		v = 0
	} else {
		//v = float32(r.TotalSpeed+r.TotalAttitude+r.TotalQuality)/float32(r.Times)/3.0 + common.Variables.HKPRankAllBase()
		v = float32(r.TotalSpeed+r.TotalAttitude+r.TotalQuality) / float32(r.Times-r.FailTimes-r.RejectTimes) / 3.0
		if v > 5 {
			v = 5
		}
	}
	return
}

func (r *HKPRank) RankAllDesc() string {
	v := r.AttitudeDesc()
	if r.QualityDesc() != "" {
		if v != "" {
			v = v + "," + r.QualityDesc()
		} else {
			v = r.QualityDesc()
		}
	}
	if r.SpeedDesc() != "" {
		if v != "" {
			v = v + "," + r.SpeedDesc()
		} else {
			v = r.SpeedDesc()
		}
	}
	return v
}

type HKPDetail struct {
	Job  HKPJob
	Rank HKPRank
}

//community->jobid->userid->job_detail映射
type UserHKPDetailMap map[common.UIDType]HKPDetail
type HKPJobUserMap map[uint]UserHKPDetailMap
type CommunityHKPJobMap map[uint]HKPJobUserMap

type UidHKPJobMap map[common.UIDType]map[uint]*HKPDetail
