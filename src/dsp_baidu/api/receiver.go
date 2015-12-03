package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
	"yf_pkg/log"
	yf_http "yf_pkg/net/http"
	"yf_pkg/service"
	"yuanfen/push/hub/config"
)

type Receiver struct {
	conf config.Config
	log  *log.MLogger
}

//广告资源的集合：map中的key的含义依次为：广告类型、高宽比、广告形式、广告点击行为类型、mime
func key(attr, height, width, tp, adct int, mime string) string {
	if width == 0 {
		return fmt.Sprintf("%v_%v_%v_%v_%v_%v", attr, height, width, tp, adct, mime)
	} else {
		return fmt.Sprintf("%v_%v_%5d_%v_%v", attr, height*10000/width, tp, adct, mime)
	}
}

var resources map[string]*SeatBidObj

func (r *Receiver) Init(env *service.Env) error {
	r.conf = env.ModuleEnv.(config.Config)
	r.log = env.Log
	go r.reload()
	test()
	return nil
}

func (r *Receiver) BidRequest(req *service.HttpRequest, res map[string]interface{}) (e error) {
	var bid BidRequest
	if e := json.Unmarshal(req.BodyRaw, &bid); e != nil {
		return e
	}
	for _, tp := range bid.Imp.Type {
		for _, adct := range bid.Imp.Adct {
			for _, mime := range bid.Imp.Mimes {
				if seatBid, ok := resources[key(bid.Imp.Attr, bid.Imp.Height, bid.Imp.Width, tp, adct, mime)]; ok {
					bidResponse := BidResponse{fmt.Sprintf("%v", rand.Int63()), bid.RequestId, *seatBid, 1}
					res["bidresponse"] = bidResponse
					return
				}
			}
		}
	}
	bidResponse := BidResponse{fmt.Sprintf("%v", rand.Int63()), bid.RequestId, SeatBidObj{}, 0}
	res["bidresponse"] = bidResponse
	return
}

func (r *Receiver) WinPrice(req *service.HttpRequest, res map[string]interface{}) (e error) {
	//TODO:需要从百度那里获取ekey和ikey，通过调用decrypt函数来解密wp字段，具体方法可以参考test()函数
	r.log.Append(fmt.Sprintf("%v__%v__%v", req.GetParam("bidid"), req.GetParam("impid"), req.GetParam("wp")), log.NOTICE)
	yf_http.Send("http", "api2.app.yuanfenba.net", "Baidu/wp", map[string]string{"bidid": req.GetParam("bidid"), "impid": req.GetParam("impid"), "wp": req.GetParam("wp")}, nil, nil, nil, 4)
	return nil
}

func (r *Receiver) reload() {
	for {
		body := []byte{}
		body, e := yf_http.Send("http", "api2.app.yuanfenba.net", "Baidu/index", nil, nil, nil, nil, 4)
		if e == nil {
			var newResources []Resource = []Resource{}
			if e = json.Unmarshal(body, &newResources); e == nil {
				tmp := map[string]*SeatBidObj{}
				for _, r := range newResources {
					k := key(r.Attr, r.Height, r.Width, r.Admt, r.Adct, r.Mime)

					seatBid, ok := tmp[k]
					if !ok {
						seatBid = &SeatBidObj{[]BidObj{r.ToBidObj()}}
						tmp[k] = seatBid
					} else {
						seatBid.Bid = append(seatBid.Bid, r.ToBidObj())
					}
				}
				resources = tmp
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

func test() {
	ekey := []byte{0xb0, 0x8c, 0x70, 0xcf, 0xbc,
		0xb0, 0xeb, 0x6c, 0xab, 0x7e, 0x82,
		0xc6, 0xb7, 0x5d, 0xa5, 0x20, 0x72,
		0xae, 0x62, 0xb2, 0xbf, 0x4b, 0x99,
		0x0b, 0xb8, 0x0a, 0x48, 0xd8, 0x14,
		0x1e, 0xec, 0x07}
	ikey := []byte{0xbf, 0x77, 0xec, 0x55, 0xc3,
		0x01, 0x30, 0xc1, 0xd8, 0xcd,
		0x18, 0x62, 0xed, 0x2a, 0x4c,
		0xd2, 0xc7, 0x6a, 0xc3, 0x3b,
		0xc0, 0xc4, 0xce, 0x8a, 0x3d,
		0x3b, 0xbd, 0x3a, 0xd5, 0x68,
		0x77, 0x92}
	websafeB64EncodedCiphertext := "SjpvRwAB4kB7jEpgW5IA8p73ew9ic6VZpFsPnA"
	b64EncodedCiphertext := unWebSafeAndPad(websafeB64EncodedCiphertext)
	decode := make([]byte, base64.StdEncoding.DecodedLen(len(b64EncodedCiphertext)))
	if _, e := base64.StdEncoding.Decode(decode, []byte(b64EncodedCiphertext)); e != nil {
		fmt.Println("base64 decode error:", e.Error())
		return
	}
	text, e := decrypt(decode, ekey, ikey)
	if e != nil {
		fmt.Println("decrypt error:", e.Error())
		return
	}
	var data int64
	b_buf := bytes.NewBuffer(text[0:8])
	binary.Read(b_buf, binary.BigEndian, &data)
	fmt.Println("decrypt data:", data)
}
