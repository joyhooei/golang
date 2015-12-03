package adview

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"yf_pkg/log"
	yf_http "yf_pkg/net/http"
	"yf_pkg/service"
	"yf_pkg/utils"
	"yuanfen/push/hub/config"
)

type Receiver struct {
	conf config.Config
	log  *log.MLogger
}

var ekey = []byte("qOsAT9wvHMg8kUZMTtvW3XYp1JsYR8iM")
var ikey = []byte("u9xYPRv1QcilarZPM1IG7eTOymtUXpru")

//广告资源的集合：map中的key的含义依次为：广告类型、高宽比
func key(instl, height, width int) string {
	if width == 0 {
		return fmt.Sprintf("%v_%v_%v", instl, height, width)
	} else {
		return fmt.Sprintf("%v_%05d", instl, height*10000/width)
	}
}

var resources map[string][]SeatBidObj

func (r *Receiver) Init(env *service.Env) error {
	r.conf = env.ModuleEnv.(config.Config)
	r.log = env.Log
	go r.reload()
	price, e := Decrypt("zayT-VABAABRLwJkbUY-YL6g-Iai1WQbE13KqQ")
	fmt.Println("price:", price, "error:", e)

	return nil
}

func (r *Receiver) BidRequest(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var bid BidRequest
	if e := json.Unmarshal(req.BodyRaw, &bid); e != nil {
		return e
	}
	for _, imp := range bid.Imp {
		if imp.Instl <= 1 { //仅支持横幅或插屏广告
			if sb, ok := resources[key(imp.Instl, imp.Banner.Height, imp.Banner.Width)]; ok {
				seatBid := make([]SeatBidObj, len(sb))
				copy(seatBid, sb)
				res["id"] = bid.RequestAdid
				res["seatbid"] = seatBid
				for i := 0; i < len(seatBid); i++ {
					bid := make([]BidObj, len(seatBid[i].Bid))
					copy(bid, seatBid[i].Bid)
					seatBid[i].Bid = bid
					for j := 0; j < len(seatBid[i].Bid); j++ {
						curl := make([]string, len(seatBid[i].Bid[j].Curl))
						copy(curl, seatBid[i].Bid[j].Curl)
						seatBid[i].Bid[j].Curl = curl
						seatBid[i].Bid[j].Impid = imp.Id
						seatBid[i].Bid[j].Wurl = fmt.Sprintf("http://dsp.yuanfenba.net:15555/adview/WinPrice?bidid=%v&impid=%v&wp=%%WIN_PRICE%%", seatBid[i].Bid[j].Cid, imp.Id)
						for k := 0; k < len(seatBid[i].Bid[j].Curl); k++ {
							seatBid[i].Bid[j].Curl[k] = strings.Replace(seatBid[i].Bid[j].Curl[k], "%%impid%%", utils.ToString(imp.Id), -1)
						}
						tmp := seatBid[i].Bid[j].Nurl
						seatBid[i].Bid[j].Nurl = make(map[string][]string, len(tmp))
						for key, _ := range tmp {
							seatBid[i].Bid[j].Nurl[key] = make([]string, len(tmp[key]))
							for k := 0; k < len(tmp[key]); k++ {
								seatBid[i].Bid[j].Nurl[key][k] = strings.Replace(tmp[key][k], "%%impid%%", utils.ToString(imp.Id), -1)
							}
						}
					}
				}
				return
			}
		}
	}
	res["id"] = bid.RequestAdid
	res["seatbid"] = []interface{}{}
	return
}

func (r *Receiver) WinPrice(req *service.HttpRequest, res map[string]interface{}) (e error) {
	price, e := Decrypt(req.GetParam("wp"))
	if e != nil {
		r.log.Append(fmt.Sprintf("%v__%v__%v:%v", req.GetParam("bidid"), req.GetParam("impid"), req.GetParam("wp"), e.Error()), log.NOTICE)
	} else {
		r.log.Append(fmt.Sprintf("%v__%v__%v:%v", req.GetParam("bidid"), req.GetParam("impid"), req.GetParam("wp"), price), log.NOTICE)
		yf_http.Send("http", "api2.app.yuanfenba.net", "Adview/wp", map[string]string{"bidid": req.GetParam("bidid"), "impid": req.GetParam("impid"), "wp": utils.ToString(price)}, nil, nil, nil, 4)
	}
	return nil
}

func (r *Receiver) reload() {
	for {
		body, e := yf_http.Send("http", "api2.app.yuanfenba.net", "adview/index", nil, nil, nil, nil, 4)
		if e == nil {
			var newResources []BidObj = []BidObj{}
			if e = json.Unmarshal(body, &newResources); e == nil {
				tmp := map[string][]SeatBidObj{}
				for _, r := range newResources {
					k := key(r.Instl, r.Height, r.Width)
					fmt.Println("key:", k)

					seatBid, ok := tmp[k]
					if !ok {
						v := SeatBidObj{[]BidObj{r}}
						tmp[k] = []SeatBidObj{v}
					} else {
						seatBid[0].Bid = append(seatBid[0].Bid, r)
					}
				}
				resources = tmp
			} else {
				r.log.Append(e.Error())
			}
		}
		if e != nil {
			r.log.Append(e.Error())
		}
		time.Sleep(3 * time.Second)
	}
}

func decrypt(ctext, ekey, ikey []byte) (text []byte, e error) {
	tlen := len(ctext) - 20
	if tlen < 0 {
		return nil, errors.New("The plain text length can't be negative.")
	}
	iv := make([]byte, 16)
	copy(iv, ctext)
	text = make([]byte, tlen)
	ctext_end := 16 + tlen
	add_iv_count_byte := true
	for ctext_begin, text_begin := 16, 0; ctext_begin < ctext_end; {
		mac := hmac.New(sha1.New, ekey)
		mac.Write(iv)
		pad := mac.Sum(nil)
		for i := 0; i < 20 && ctext_begin < ctext_end; i++ {
			text[text_begin] = ctext[ctext_begin] ^ pad[i]
			text_begin++
			ctext_begin++
		}
		if !add_iv_count_byte {
			index := len(iv) - 1
			iv[index] += 1
			add_iv_count_byte = (iv[index] == 0)
		}
		if add_iv_count_byte {
			add_iv_count_byte = false
			iv = append(iv, byte(0))
		}
	}
	return
}

func unWebSafeAndPad(webSafe string) string {
	pad := ""
	if (len(webSafe) % 4) == 2 {
		pad = "=="
	} else if (len(webSafe) % 4) == 3 {
		pad = "="
	}
	return strings.Replace(strings.Replace(webSafe, "-", "+", -1), "_", "/", -1) + pad
}

func Decrypt(websafeB64EncodedCiphertext string) (price int64, e error) {
	b64EncodedCiphertext := unWebSafeAndPad(websafeB64EncodedCiphertext)
	decode := make([]byte, base64.StdEncoding.DecodedLen(len(b64EncodedCiphertext)))
	if _, e := base64.StdEncoding.Decode(decode, []byte(b64EncodedCiphertext)); e != nil {
		return 0, errors.New(fmt.Sprintln("base64 decode error:", e.Error()))
	}
	text, e := decrypt(decode, ekey, ikey)
	if e != nil {
		return 0, errors.New(fmt.Sprintln("decrypt error:", e.Error()))
	}
	var data int64
	b_buf := bytes.NewBuffer(text[0:8])
	binary.Read(b_buf, binary.BigEndian, &data)
	return data, nil
}
