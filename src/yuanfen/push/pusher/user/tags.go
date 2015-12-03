package user

import "sync"

type TagUserMap struct {
	data map[string]map[uint32]bool
	lock sync.RWMutex
}

func NewTagUserMap(capacity ...int) *TagUserMap {
	var tm TagUserMap
	if len(capacity) >= 1 {
		tm.data = make(map[string]map[uint32]bool, capacity[0])
	} else {
		tm.data = make(map[string]map[uint32]bool)
	}
	return &tm
}

func (tm *TagUserMap) AddUserTag(uid uint32, tag string) {
	tm.lock.Lock()
	users, ok := tm.data[tag]
	if !ok {
		tm.data[tag] = map[uint32]bool{uid: true}
	} else {
		users[uid] = true
	}
	tm.lock.Unlock()
}
func (tm *TagUserMap) DelUserTag(uid uint32, tag string) {
	tm.lock.Lock()
	users, ok := tm.data[tag]
	if ok {
		delete(users, uid)
	}
	tm.lock.Unlock()
}

func (tm *TagUserMap) AddUserTags(uid uint32, tags map[string]bool) {
	tm.lock.Lock()
	for tag, _ := range tags {
		users, ok := tm.data[tag]
		if !ok {
			users = map[uint32]bool{uid: true}
			tm.data[tag] = users
		} else {
			users[uid] = true
		}
	}
	tm.lock.Unlock()
}

func (tm *TagUserMap) DelUserTags(uid uint32, tags map[string]bool) {
	tm.lock.Lock()
	for tag, _ := range tags {
		users, ok := tm.data[tag]
		if ok {
			delete(users, uid)
		}
	}
	tm.lock.Unlock()
}

//遍历某个tag下的所有uid
func (tm *TagUserMap) Iterate(tag string, iter func(uid uint32) error) error {
	tm.lock.RLock()
	defer tm.lock.RUnlock()
	for k, _ := range tm.data[tag] {
		if e := iter(k); e != nil {
			return e
		}
	}
	return nil
}
