package util

import (
	"sync/atomic"
)

type AtomicCounter struct {
	counter int64
}

func (ac *AtomicCounter) Increment() {
	atomic.AddInt64(&ac.counter, 1)
}

func (ac *AtomicCounter) Decrement() {
	atomic.AddInt64(&ac.counter, -1)
}

func (ac *AtomicCounter) GetCount() int64 {
	return atomic.LoadInt64(&ac.counter)
}

func (ac *AtomicCounter) Reset() {
	atomic.StoreInt64(&ac.counter, 0)
}
