package award

type Award struct {
	Id       uint32 `json:"id"`
	Price    uint32 `json:"price"`
	Name     string `json:"name"`
	Atype    uint32 `json:"type"`
	Img      string `json:"img"`
	Info     string `json:"info"`
	Unit     string `json:"unit"`
	Game_img string `json:"game_img"`
	PushFlag int    `json:"_"`
	ShowType int    `json:"show_type"`
}
