/*
线程安全的map。

示例：
	var safeMap SafeMap = New()

	safeMap.Set(13, 14)
	safeMap.Set(15, "hello")
	e := safeMap.Iterate(func(key interface{}, value interface{}) error {
		fmt.Println(key, value)
		return nil
	})
	if e != nil {
		fmt.Println(e.Error())
	}
*/
package safe_map

import "sync"

type SafeMap struct {
	data map[interface{}]interface{}
	lock sync.RWMutex
}

func New(capacity ...int) *SafeMap {
	var sm SafeMap
	if len(capacity) >= 1 {
		sm.data = make(map[interface{}]interface{}, capacity[0])
	} else {
		sm.data = make(map[interface{}]interface{})
	}
	return &sm
}

func (sm *SafeMap) Set(key interface{}, value interface{}) {
	sm.lock.Lock()
	sm.data[key] = value
	sm.lock.Unlock()
}

func (sm *SafeMap) Len() (length int) {
	sm.lock.RLock()
	length = len(sm.data)
	sm.lock.RUnlock()
	return
}

func (sm *SafeMap) Get(key interface{}) (value interface{}, exist bool) {
	sm.lock.RLock()
	value, exist = sm.data[key]
	sm.lock.RUnlock()
	return
}

func (sm *SafeMap) Del(key interface{}) {
	sm.lock.Lock()
	delete(sm.data, key)
	sm.lock.Unlock()
}

//遍历SafeMap中的所有元素
func (sm *SafeMap) Iterate(iter func(key interface{}, value interface{}) error) error {
	sm.lock.RLock()
	defer sm.lock.RUnlock()
	for k, v := range sm.data {
		if e := iter(k, v); e != nil {
			return e
		}
	}
	return nil
}

//获取某个元素并做一些处理
func (sm *SafeMap) GetAndDo(key interface{}, do func(value interface{}, exist bool)) {
	sm.lock.RLock()
	v, ok := sm.data[key]
	do(v, ok)
	sm.lock.RUnlock()
	return
}
