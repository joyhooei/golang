package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
	"yf_pkg/log"
	"yf_pkg/utils"
	"yuanfen/push/pusher/db"
)

var urls []string
var logger *log.Logger
var routines int
var httpClient http.Client

func init() {
	urls = make([]string, 0, 10)
	logger, _ = log.New("/dev/null", 10000, log.ERROR)
	httpClient.Timeout = 2 * time.Second
}

func Init(l *log.Logger) error {
	logger = l
	_, e := update()
	go updateLoop()
	return e
}

func updateLoop() {
	for {
		time.Sleep(10 * time.Second)
		update()
	}
}

func update() ([]string, error) {
	newUrls, e := db.GetNotificationUrls()
	if e != nil {
		return nil, e
	}
	urls = newUrls
	logger.Append(fmt.Sprintf("refresh notification urls : %v", urls), log.NOTICE)
	return urls, nil
}

func NotifyOffline(uid uint32) {
	j, _ := json.Marshal(map[string]interface{}{"type": "offline", "uid": uid})
	send(j)
}

func NotifyLocation(uid uint32, lat float64, lng float64) {
	j, _ := json.Marshal(map[string]interface{}{"type": "location", "uid": uid, "lat": lat, "lng": lng})
	send(j)

}
func send(data []byte) {
	routines++
	for _, url := range urls {
		start := time.Now()
		logger.Append(fmt.Sprintf("notify [%v]: %v routines %v", url, string(data), routines), log.DEBUG)
		resp, e := httpClient.Post(url, "text/html", bytes.NewBuffer(data))
		if e != nil {
			logger.Append(fmt.Sprintf("notify [%v]:%v error : %v", url, string(data), e.Error()), log.ERROR)
			continue
		}
		logger.Append(fmt.Sprintf("post [%v]:%v success", url, string(data)), log.DEBUG)
		if resp.Body != nil {
			_, e = ioutil.ReadAll(resp.Body)
			logger.Append(fmt.Sprintf("ReadAll [%v] finished", url), log.DEBUG)
			resp.Body.Close()
			if e != nil {
				logger.Append(fmt.Sprintf("notify [%v] error : %v", url, e.Error()), log.ERROR)
			}
		} else {
			logger.Append(fmt.Sprintf("response body [%v] is nil", url), log.DEBUG)
		}
		utils.PrintDuration(fmt.Sprintf("send %v", url), start, time.Second)
	}
	routines--
}
