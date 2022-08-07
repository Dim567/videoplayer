package multithread

import (
	"sync"
	"time"
)

type SharedBuffer struct {
	queue    []interface{}
	capacity int
	opened   bool
	mu       sync.Mutex
}

func (sb *SharedBuffer) Read() (interface{}, bool) {
	var item interface{}
	var bufOpened bool
	var bufSize int
	var continueLoop = true
	var shouldSleep = false
	for continueLoop {
		if shouldSleep {
			time.Sleep(time.Millisecond)
			shouldSleep = false
		}
		sb.mu.Lock()
		bufOpened = sb.opened
		bufSize = len(sb.queue)

		switch {
		case bufSize > 1:
			item = sb.queue[0]
			sb.queue = sb.queue[1:]
			continueLoop = false
		case bufSize == 1:
			item = sb.queue[0]
			if bufOpened {
				sb.queue = make([]interface{}, 0, sb.capacity)
			} else {
				sb.queue = []interface{}{}
			}
			continueLoop = false
		case bufSize == 0 && !bufOpened:
			continueLoop = false
		default:
			shouldSleep = true
		}
		sb.mu.Unlock()
	}
	return item, bufOpened
}

func (sb *SharedBuffer) Write(elem interface{}) {
	var bufClosed bool
	var bufSize int
	var continueLoop = true
	var shouldSleep = false
	for continueLoop {
		if shouldSleep {
			time.Sleep(time.Millisecond)
			shouldSleep = false
		}
		sb.mu.Lock()
		bufClosed = !sb.opened
		bufSize = len(sb.queue)

		switch {
		case bufClosed:
			continueLoop = false
		case bufSize != sb.capacity:
			sb.queue = append(sb.queue, elem)
			continueLoop = false
		default:
			shouldSleep = true
		}
		sb.mu.Unlock()
	}
}

func (sb *SharedBuffer) Size() int {
	sb.mu.Lock()
	size := len(sb.queue)
	sb.mu.Unlock()
	return size
}

func (sb *SharedBuffer) Close() {
	sb.mu.Lock()
	sb.opened = false
	sb.mu.Unlock()
}

func (sb *SharedBuffer) Purge() {
	sb.mu.Lock()
	sb.queue = make([]interface{}, 0, sb.capacity)
	sb.mu.Unlock()
}

func NewSharedBuffer(capacity int) *SharedBuffer {
	var sb = SharedBuffer{
		queue:    make([]interface{}, 0, capacity),
		capacity: capacity,
		opened:   true,
	}
	return &sb
}
