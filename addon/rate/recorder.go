package rate

import (
	"sync"
	"time"
)

const (
	heartbeatDeadline = 5 * time.Minute
)

type Traffic struct {
	Sent uint64
	Rcvd uint64
}

type Record struct {
	IP    IP
	Usage Traffic
}

// RateTracker 流量跟踪器
type Recorder struct {
	startTime     time.Time
	lastHeartbeat time.Time
	ip            IP

	// 总量统计
	total Traffic
	// 瞬时统计
	unrecorded Traffic

	// channel驱动
	sendChan chan uint64
	recvChan chan uint64
	wg       sync.WaitGroup

	// heartbeat
	stopChan chan struct{}
}

// NewRateTracker 创建新的跟踪器
func newRateRecorder(ip IP) *Recorder {
	now := time.Now()
	rt := &Recorder{
		startTime: now,
		ip:        ip,
		sendChan:  make(chan uint64, 1000), // 缓冲channel
		recvChan:  make(chan uint64, 1000), // 缓冲channel
		stopChan:  make(chan struct{}),
	}

	// 启动自动记录协程
	rt.wg.Add(1)
	go rt.recordLoop()

	return rt
}

// SendChan 获取发送channel
func (rt *Recorder) SendChan() chan<- uint64 {
	return rt.sendChan
}

// RecvChan 获取接收channel
func (rt *Recorder) RecvChan() chan<- uint64 {
	return rt.recvChan
}

func (rt *Recorder) Record() Record {
	var traffic Traffic
	traffic.Sent = rt.unrecorded.Sent
	traffic.Rcvd = rt.unrecorded.Rcvd
	rt.unrecorded.Sent = 0
	rt.unrecorded.Rcvd = 0
	rt.lastHeartbeat = time.Now()
	return Record{IP: rt.ip, Usage: traffic}
}

// recordLoop 自动记录循环
func (rt *Recorder) recordLoop() {
	defer rt.wg.Done()

	for {
		select {
		case sent := <-rt.sendChan:
			// 更新发送量
			rt.total.Sent += sent
			rt.unrecorded.Sent += sent
			rt.lastHeartbeat = time.Now()

		case received := <-rt.recvChan:
			// 更新接收量
			rt.total.Rcvd += received
			rt.unrecorded.Rcvd += received
			rt.lastHeartbeat = time.Now()

		case <-rt.stopChan:
			return
		}
	}
}

// Stop 停止跟踪器
func (rt *Recorder) Stop() {
	close(rt.stopChan)
	rt.wg.Wait()
}

type Stats struct {
	TotalSent       uint64 `json:"total_sent"`
	TotalReceived   uint64 `json:"total_received"`
	CurrentSent     uint64 `json:"current_sent"`
	CurrentReceived uint64 `json:"current_received"`

	TotalSentBps       float64 `json:"total_sent_bps"`
	TotalReceivedBps   float64 `json:"total_received_bps"`
	CurrentSentBps     float64 `json:"current_sent_bps"`
	CurrentReceivedBps float64 `json:"current_received_bps"`
}

// GetStats 获取所有统计信息
func (rt *Recorder) GetStats() Stats {
	secondTillNow := time.Since(rt.startTime).Seconds()

	return Stats{
		// 总量统计
		TotalSent:     rt.total.Sent,
		TotalReceived: rt.total.Rcvd,
		// 当前窗口统计
		CurrentSent:     rt.unrecorded.Sent,
		CurrentReceived: rt.unrecorded.Rcvd,
		// 当前窗口bps
		CurrentSentBps:     float64(rt.unrecorded.Sent) / secondTillNow,
		CurrentReceivedBps: float64(rt.unrecorded.Rcvd) / secondTillNow,
		// 总量bps
		TotalSentBps:     float64(rt.total.Sent) / secondTillNow,
		TotalReceivedBps: float64(rt.total.Rcvd) / secondTillNow,
	}
}
