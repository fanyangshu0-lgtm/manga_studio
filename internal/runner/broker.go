package runner

import (
	"sync"

	"manga-drama-studio/internal/domain"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan domain.RunEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{subscribers: map[string]map[chan domain.RunEvent]struct{}{}}
}

func (b *Broker) Subscribe(runID string) (<-chan domain.RunEvent, func()) {
	channel := make(chan domain.RunEvent, 24)
	b.mu.Lock()
	if b.subscribers[runID] == nil {
		b.subscribers[runID] = map[chan domain.RunEvent]struct{}{}
	}
	b.subscribers[runID][channel] = struct{}{}
	b.mu.Unlock()
	return channel, func() {
		b.mu.Lock()
		if _, ok := b.subscribers[runID][channel]; ok {
			delete(b.subscribers[runID], channel)
			close(channel)
		}
		if len(b.subscribers[runID]) == 0 {
			delete(b.subscribers, runID)
		}
		b.mu.Unlock()
	}
}

func (b *Broker) Publish(event domain.RunEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for channel := range b.subscribers[event.RunID] {
		select {
		case channel <- event:
		default:
		}
	}
}
