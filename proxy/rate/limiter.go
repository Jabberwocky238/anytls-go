package rate

import "time"

// Limiter 是一个简单的令牌桶限速器
const (
	LimitBps = 5_000_000 // 1MBps
)

type Limiter struct {
	limitBps uint64
}

// NewLimiter 创建一个新的限速器
func newLimiter() *Limiter {
	return &Limiter{
		limitBps: LimitBps,
	}
}

// Allow 判断是否允许通过
func (l *Limiter) Disallow(currentBps uint64) bool {
	return currentBps > l.limitBps
}

func (l *Limiter) TryLimitSend(recorder *Recorder) {
	if l.Disallow(recorder.getStats().CurrentSent) {
		time.Sleep(time.Millisecond * 100)
	}
}

func (l *Limiter) TryLimitRecv(recorder *Recorder) {
	if l.Disallow(recorder.getStats().CurrentReceived) {
		time.Sleep(time.Millisecond * 100)
	}
}
