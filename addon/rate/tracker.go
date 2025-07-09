package rate

import (
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// IPTracker a bucket of *RateTracker, indexed by IP
type IPTracker struct {
	recorders map[IP]*Recorder
	mu        sync.RWMutex
}

var Tracker = newIPTracker()

// NewIPTracker creates a new IPTracker
func newIPTracker() *IPTracker {
	return &IPTracker{
		recorders: make(map[IP]*Recorder),
	}
}

// WithIP gets or creates a tracker by IP
func (b *IPTracker) WithIP(addr net.Addr) *Recorder {
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

func (b *IPTracker) Clean() {
	b.mu.Lock()
	defer b.mu.Unlock()
	total := len(b.recorders)
	remain := 0
	for ip, t := range b.recorders {
		if time.Since(t.lastHeartbeat) > heartbeatDeadline {
			logrus.Infof("[Rate] stop recorder %s", ip)
			t.Stop()
			delete(b.recorders, ip)
		} else {
			remain++
		}
	}
	logrus.Infof("[Rate] clean %d recorders, remain %d", total-remain, remain)
}

func (b *IPTracker) Records() []Record {
	var feedbacks []Record
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, t := range b.recorders {
		feedbacks = append(feedbacks, t.Record())
	}
	return feedbacks
}
