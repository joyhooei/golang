package common

type ScoreElem struct {
	Uid      UIDType
	Comm     uint
	Rank     int
	Priority int32 //随机生成的优先级
}
type ScoreElems []ScoreElem
type RandomScoreElems []ScoreElem

func (a ScoreElems) Len() int           { return len(a) }
func (a ScoreElems) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ScoreElems) Less(i, j int) bool { return a[i].Rank > a[j].Rank }

func (a RandomScoreElems) Len() int           { return len(a) }
func (a RandomScoreElems) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RandomScoreElems) Less(i, j int) bool { return a[i].Priority > a[j].Priority }
