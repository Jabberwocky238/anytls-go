package ratetracker

import (
	"net"
	"sync"
)

type IP = string

func ip(addr net.Addr) IP {
	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return ""
	}
	return ip
}

// IPBucket a bucket of *RateTracker, indexed by IP
type IPBucket struct {
	recorders map[IP]*RateRecorder
	mu        sync.RWMutex
}

// NewIPBucket creates a new IPBucket
func newIPBucket() *IPBucket {
	return &IPBucket{
		recorders: make(map[IP]*RateRecorder),
	}
}

// GetRecorder gets or creates a tracker by IP
func (b *IPBucket) GetRecorder(addr net.Addr) *RateRecorder {
	b.mu.RLock()
	ip := ip(addr)
	tracker, ok := b.recorders[ip]
	b.mu.RUnlock()

	if ok {
		return tracker
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	tracker, ok = b.recorders[ip]
	if ok {
		return tracker
	}

	tracker = newRateRecorder(ip)
	b.recorders[ip] = tracker
	return tracker
}

// RemoveRecorder removes a tracker by IP
func (b *IPBucket) RemoveRecorder(addr net.Addr) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ip := ip(addr)
	if tracker, ok := b.recorders[ip]; ok {
		tracker.Stop()
		delete(b.recorders, ip)
	}
}
