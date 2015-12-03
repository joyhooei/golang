package user

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	"yf_pkg/utils"
)

var keys map[uint32]*Key
var klock sync.RWMutex

type Key struct {
	//	Uid     uint32
	Key     uint32
	Timeout int64
}

func init() {
	keys = make(map[uint32]*Key)
	go removeTimeoutkeys()
}
func AddKey(uid uint32) (key *Key) {
	k := &Key{rand.Uint32(), utils.Now.Unix() + KEY_TIMEOUT}
	klock.RLock()
	keys[uid] = k
	klock.RUnlock()
	return k
}

func DelKey(uid uint32) {
	klock.Lock()
	delete(keys, uid)
	klock.Unlock()
}
func IsValid(uid uint32, key uint32) (valid bool, reason string) {
	klock.RLock()
	defer klock.RUnlock()
	k, ok := keys[uid]
	if ok {
		if key != k.Key {
			reason = "invalid key"
		} else if utils.Now.Unix() > k.Timeout {
			reason = "timeout"
		} else {
			return true, ""
		}
	} else {
		reason = "not found user key"
	}
	return false, reason
}

func removeTimeoutkeys() {
	for {
		klock.Lock()
		for uid, key := range keys {
			if utils.Now.Unix() > key.Timeout {
				fmt.Printf("delete key <%v,%v>\n", uid, key)
				delete(keys, uid)
			}
		}
		klock.Unlock()
		time.Sleep(10 * time.Second)
	}
}
