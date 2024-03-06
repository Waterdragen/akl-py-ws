package main

import (
	"sync"

	genkey "github.com/waterdragen/akl-ws/genkey"
)

type ConnUsers struct {
	mu   sync.RWMutex
	data map[uint64]*genkey.UserData
}

func NewConnUsers() *ConnUsers {
	return &ConnUsers{
		data: make(map[uint64]*genkey.UserData),
	}
}

func (cu *ConnUsers) Add(key uint64, value *genkey.UserData) {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	cu.data[key] = value
}

func (cu *ConnUsers) Pop(key uint64) {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	delete(cu.data, key)
}

func (cu *ConnUsers) Get(key uint64) (*genkey.UserData, bool) {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	value, ok := cu.data[key]
	return value, ok
}
