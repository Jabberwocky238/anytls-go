package rate

import "time"

// Limiter 是一个简单的令牌桶限速器
const (
	B        = 1
	KB       = 1024 * B
	MB       = 1024 * KB
	LimitBps = 100 * MB // 10MBps
)

type Limiter struct {
	limitBps float64
}

// NewLimiter 创建一个新的限速器
func newLimiter() *Limiter {
	return &Limiter{
		limitBps: LimitBps,
	}
}

// Allow 判断是否允许通过
func (l *Limiter) Disallow(currentBps float64) bool {
	return currentBps > l.limitBps
}

func (l *Limiter) TryLimitSend(recorder *Recorder) {
	if l.Disallow(recorder.getStats().CurrentSentBps) {
		time.Sleep(time.Millisecond * 100)
	}
}

func (l *Limiter) TryLimitRecv(recorder *Recorder) {
	if l.Disallow(recorder.getStats().CurrentReceivedBps) {
		time.Sleep(time.Millisecond * 100)
	}
}
