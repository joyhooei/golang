package api

type DeviceObj struct {
	DeviceType int `json:"devicetype"` //设备类型：0-安卓手机，1-安卓平板，2-iphone，3-ipad，4-其它
}

type ImpObj struct {
	ImpId    string   `json:"impid"`    //维度ID
	Width    int      `json:"w"`        //广告宽度
	Height   int      `json:"h"`        //广告高度
	Attr     int      `json:"attr"`     //广告类型
	Type     []int    `json:"type"`     //广告形式
	Adct     []int    `json:"adct"`     //广告点击行为类型
	Mimes    []string `json:"mimes"`    //支持的图片类型
	BidFloor int      `json:"bidfloor"` //底价
}

type BidRequest struct {
	RequestId string    `json:"requestid"`
	Device    DeviceObj `json:"device"`
	Imp       ImpObj    `json:"imp"`
}

type BidObj struct {
	Id      string   `json:"id"`
	Impid   string   `json:"impid"`
	St      int      `json:"st"`
	Price   int      `json:"price"`
	Attr    int      `json:"attr"`
	Admt    int      `json:"admt"`
	Adm     string   `json:"adm"`
	Adi     string   `json:"adi"`
	Adicon  string   `json:"adicon"`
	Adt     string   `json:"adt"`
	Ads     string   `json:"ads"`
	Cid     string   `json:"cid"`
	Crid    string   `json:"crid"`
	Cat     int      `json:"cat"`
	Height  int      `json:"h"`
	Width   int      `json:"w"`
	Adct    int      `json:"adct"`
	Bundle  string   `json:"bundle"`
	Adtm    int      `json:"adtm"`
	Nurl    string   `json:"nurl"`
	Adcurl  string   `json:"adcurl"`
	Iurl    []string `json:"iurl"`
	Curl    []string `json:"curl"`
	Adomain string   `json:"adomain"`
}

type SeatBidObj struct {
	Bid []BidObj `json:"bid"`
}

type BidResponse struct {
	BidId     string     `json:"bidid"`
	RequestId string     `json:"requestid"`
	SeatBid   SeatBidObj `json:"seatbid"`
	Nbr       int        `json:"nbr"`
}

type Resource struct {
	Id      string   `json:"id"`
	Impid   string   `json:"impid"`
	Mime    string   `json:"mimes"`
	St      int      `json:"st"`
	Price   int      `json:"price"`
	Attr    int      `json:"attr"`
	Admt    int      `json:"admt"`
	Adm     string   `json:"adm"`
	Adi     string   `json:"adi"`
	Adicon  string   `json:"adicon"`
	Adt     string   `json:"adt"`
	Ads     string   `json:"ads"`
	Cid     string   `json:"cid"`
	Crid    string   `json:"crid"`
	Cat     int      `json:"cat"`
	Height  int      `json:"h"`
	Width   int      `json:"w"`
	Adct    int      `json:"adct"`
	Bundle  string   `json:"bundle"`
	Adtm    int      `json:"adtm"`
	Nurl    string   `json:"nurl"`
	Adcurl  string   `json:"adcurl"`
	Iurl    []string `json:"iurl"`
	Curl    []string `json:"curl"`
	Adomain string   `json:"adomain"`
}

func (r *Resource) ToBidObj() (bid BidObj) {
	bid.Id, bid.Impid, bid.St, bid.Price, bid.Attr, bid.Admt, bid.Adm, bid.Adi, bid.Adicon, bid.Adt, bid.Ads, bid.Cid, bid.Crid = r.Id, r.Impid, r.St, r.Price, r.Attr, r.Admt, r.Adm, r.Adi, r.Adicon, r.Adt, r.Ads, r.Cid, r.Crid
	bid.Cat, bid.Height, bid.Width, bid.Adct, bid.Bundle, bid.Adtm, bid.Adcurl, bid.Iurl, bid.Curl, bid.Adomain = r.Cat, r.Height, r.Width, r.Adct, r.Bundle, r.Adtm, r.Adcurl, r.Iurl, r.Curl, r.Adomain
	bid.Nurl = "http://dsp.yuanfenba.net/dsp/WinPrice?bidid=%%bidid%%&impid=%%impid%%&wp=%%wp%%"
	return
}
