package manager

import (
	"sync"
)

// EventBroadcaster broadcasts events to multiple subscribers
type EventBroadcaster struct {
	subscribers map[chan interface{}]struct{}
	mu          sync.RWMutex
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster() *EventBroadcaster {
	return &EventBroadcaster{
		subscribers: make(map[chan interface{}]struct{}),
	}
}

// Subscribe adds a new subscriber
func (eb *EventBroadcaster) Subscribe() chan interface{} {
	ch := make(chan interface{}, 100)
	eb.mu.Lock()
	eb.subscribers[ch] = struct{}{}
	eb.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber
func (eb *EventBroadcaster) Unsubscribe(ch chan interface{}) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	if _, ok := eb.subscribers[ch]; ok {
		delete(eb.subscribers, ch)
		close(ch)
	}
}

// Broadcast sends an event to all subscribers
func (eb *EventBroadcaster) Broadcast(event interface{}) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	for ch := range eb.subscribers {
		select {
		case ch <- event:
		default:
			// Channel is full, skip this subscriber
		}
	}
}
