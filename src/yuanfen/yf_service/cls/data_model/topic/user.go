package topic

type User struct {
	Uid      uint32 `json:"uid"`
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
	Age      int    `json:"age"`
	Avatar   string `json:"avatar"`
}
