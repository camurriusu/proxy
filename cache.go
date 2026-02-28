package main

import "sync"

type SafeCache struct {
	mutex sync.RWMutex
	data  map[string][]byte
}

func (sf *SafeCache) Get(key string) ([]byte, bool) {
	sf.mutex.RLock()
	defer sf.mutex.RUnlock()
	value, found := sf.data[key]
	return value, found
}

func (sf *SafeCache) Set(key string, value []byte) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.data[key] = value
}

var cache = &SafeCache{
	data: make(map[string][]byte),
}
