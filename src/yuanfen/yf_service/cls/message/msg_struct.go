package message

import "time"

type LocationChange struct {
	Uid uint32
	Lat float64
	Lng float64
}

type Online struct {
	Uid uint32
}

type RecommendChange struct {
	Uid uint32
}

type Offline struct {
	Uid uint32
}

type Register struct {
	Uid      uint32
	Gender   int
	Birthday time.Time
}

type Visit struct {
	Vistor uint32
	Target uint32
}

type BirthdayChange struct {
	Uid      uint32
	Birthday time.Time
}

type MsgDanger struct {
	Uid     uint32
	Content string
}

type ClearCache struct {
	Uid uint32
}

type CreateTopic struct {
	Uid uint32
	Tid uint32
}

type OnTop struct {
	Uid  uint32
	Stat int //0为离线 1为在线
}
