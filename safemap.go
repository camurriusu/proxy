package main

import "sync"

type SafeMap struct {
	mutex sync.RWMutex
	data  map[string]any
}

func (sf *SafeMap) Get(key string) (any, bool) {
	sf.mutex.RLock()
	defer sf.mutex.RUnlock()
	value, found := sf.data[key]
	return value, found
}

func (sf *SafeMap) Set(key string, value any) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()
	sf.data[key] = value
}

var cache = &SafeMap{
	data: make(map[string]any),
}

var blocklist = &SafeMap{
	data: make(map[string]any),
}
