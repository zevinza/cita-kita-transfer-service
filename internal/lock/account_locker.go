package lock

import "sync"

// AccountLocker serializes in-process work on the same account pair.
type AccountLocker interface {
	Lock(fromID, toID string)
	Unlock(fromID, toID string)
}

type accountLocker struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewAccountLocker() AccountLocker {
	return &accountLocker{
		locks: make(map[string]*sync.Mutex),
	}
}

func (l *accountLocker) Lock(fromID, toID string) {
	first, second := sortedPair(fromID, toID)
	l.lockOne(first)
	l.lockOne(second)
}

func (l *accountLocker) Unlock(fromID, toID string) {
	first, second := sortedPair(fromID, toID)
	l.unlockOne(second)
	l.unlockOne(first)
}

func (l *accountLocker) lockOne(id string) {
	l.mu.Lock()
	m, ok := l.locks[id]
	if !ok {
		m = &sync.Mutex{}
		l.locks[id] = m
	}
	l.mu.Unlock()
	m.Lock()
}

func (l *accountLocker) unlockOne(id string) {
	l.mu.Lock()
	m, ok := l.locks[id]
	l.mu.Unlock()
	if ok {
		m.Unlock()
	}
}

func sortedPair(a, b string) (string, string) {
	if a > b {
		return b, a
	}
	return a, b
}
