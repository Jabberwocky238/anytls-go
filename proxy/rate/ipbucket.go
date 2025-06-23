package rate

import (
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
	recorders map[IP]*Recorder
	mu        sync.RWMutex
}

// NewIPBucket creates a new IPBucket
func newIPBucket() *IPBucket {
	return &IPBucket{
		recorders: make(map[IP]*Recorder),
	}
}

// GetRecorder gets or creates a tracker by IP
func (b *IPBucket) GetRecorder(addr net.Addr) *Recorder {
	b.mu.RLock()
	ip := ip(addr)
	tracker, ok := b.recorders[ip]
	b.mu.RUnlock()

	if ok {
		tracker.lastHeartbeat = time.Now()
		return tracker
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	tracker, ok = b.recorders[ip]
	if ok {
		return tracker
	}

	tracker = newRateRecorder(ip)
	tracker.lastHeartbeat = time.Now()
	b.recorders[ip] = tracker
	return tracker
}

func (b *IPBucket) Clean() {
	b.mu.Lock()
	defer b.mu.Unlock()
	total := len(b.recorders)
	remain := 0
	for ip, t := range b.recorders {
		if time.Since(t.lastHeartbeat) > heartbeatDeadline {
			delete(b.recorders, ip)
		} else {
			remain++
		}
	}
	logrus.Infof("[Rate] clean %d recorders, remain %d", total-remain, remain)
}
