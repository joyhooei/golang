package adview

type DeviceObj struct {
	DeviceType int `json:"devicetype"` //设备类型：0-未知，1-iPhone，2-android手机，3-ipad，4-wp，5-android平板，6-智能TV
}

type BannerObj struct {
	Width  int `json:"w"` //广告宽度
	Height int `json:"h"` //广告高度
}

type ImpObj struct {
	Id       string    `json:"id"` //维度ID
	Banner   BannerObj `json:"banner"`
	BidFloor int       `json:"bidfloor"` //底价
	Instl    int       `json:"instl"`    //0-横幅广告，1-插屏或全插屏广告，4-开屏，6-原生广告
}

type BidRequest struct {
	RequestAdid string    `json:"id"`
	Device      DeviceObj `json:"device"`
	Imp         []ImpObj  `json:"imp"`
}

type BidObj struct {
	Adid    string `json:"adid"`
	Impid   string `json:"impid"`
	Instl   int    `json:"instl"`
	Price   int    `json:"price"`
	Paymode int    `json:"paymode"`
	Adct    int    `json:"adct"`
	Attr    int    `json:"attr"`
	Admt    int    `json:"admt"`
	Adm     string `json:"adm"`
	Adi     string `json:"adi"`
	Adt     string `json:"adt"`
	Ads     string `json:"ads"`
	Cid     string `json:"cid"`
	Crid    string `json:"crid"`
	//Cat     int                 `json:"cat"`
	Height int `json:"adh"`
	Width  int `json:"adw"`
	//	Adtm    int                 `json:"adtm"`
	Wurl    string              `json:"wurl"`
	Adurl   string              `json:"adurl"`
	Nurl    map[string][]string `json:"nurl"`
	Curl    []string            `json:"curl"`
	Adomain string              `json:"adomain"`
}

type SeatBidObj struct {
	Bid []BidObj `json:"bid"`
}

type BidResponse struct {
	BidAdid string       `json:"id"`
	SeatBid []SeatBidObj `json:"seatbid"`
}
