package auth

import (
	"sync"
	"time"
)

type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	fails  map[string][]time.Time
	now    func() time.Time
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{
		max:    max,
		window: window,
		fails:  make(map[string][]time.Time),
		now:    time.Now,
	}
}

func (l *Limiter) prune(key string) {
	cutoff := l.now().Add(-l.window)
	kept := l.fails[key][:0]
	for _, at := range l.fails[key] {
		if at.After(cutoff) {
			kept = append(kept, at)
		}
	}
	if len(kept) == 0 {
		delete(l.fails, key)
		return
	}
	l.fails[key] = kept
}

func (l *Limiter) Blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(key)
	return len(l.fails[key]) >= l.max
}

func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(key)
	l.fails[key] = append(l.fails[key], l.now())
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}
