package rate

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	// "github.com/sirupsen/logrus"
)

const (
	windowPeriodSize  = 100 * time.Millisecond
	WindowQueueSize   = 10
	heartbeatDeadline = 5 * time.Minute
)

// RateTracker 流量跟踪器
type Recorder struct {
	startTime     time.Time
	lastHeartbeat time.Time
	ip            IP

	// 总量统计
	totalSent     atomic.Uint64
	totalReceived atomic.Uint64

	// 瞬时统计
	windowSent     atomic.Uint64
	windowReceived atomic.Uint64

	// 滑动窗口
	windowSendQueue []uint64
	windowRecvQueue []uint64
	windowIndex     int

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
		startTime:       now,
		ip:              ip,
		sendChan:        make(chan uint64, 1000), // 缓冲channel
		recvChan:        make(chan uint64, 1000), // 缓冲channel
		windowSendQueue: make([]uint64, WindowQueueSize),
		windowRecvQueue: make([]uint64, WindowQueueSize),
		stopChan:        make(chan struct{}),
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

// recordLoop 自动记录循环
func (rt *Recorder) recordLoop() {
	defer rt.wg.Done()

	ticker := time.NewTicker(windowPeriodSize)
	// printer := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case sent := <-rt.sendChan:
			// 更新发送量
			rt.totalSent.Add(sent)
			rt.windowSent.Add(sent)
			rt.lastHeartbeat = time.Now()

		case received := <-rt.recvChan:
			// 更新接收量
			rt.totalReceived.Add(received)
			rt.windowReceived.Add(received)
			rt.lastHeartbeat = time.Now()

		case <-ticker.C:
			// 每100ms重置窗口
			rt.windowSendQueue[rt.windowIndex] = rt.windowSent.Load()
			rt.windowRecvQueue[rt.windowIndex] = rt.windowReceived.Load()
			rt.windowIndex = (rt.windowIndex + 1) % WindowQueueSize
			rt.windowSent.Store(0)
			rt.windowReceived.Store(0)

		// case <-printer.C:
		// 	logrus.Infof(rt.print())

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
func (rt *Recorder) getStats() Stats {
	secondTillNow := time.Since(rt.startTime).Seconds()

	return Stats{
		// 总量统计
		TotalSent:     rt.totalSent.Load(),
		TotalReceived: rt.totalReceived.Load(),
		// 当前窗口统计
		CurrentSent:     rt.windowSent.Load(),
		CurrentReceived: rt.windowReceived.Load(),
		// 当前窗口bps
		CurrentSentBps:     float64(sum(rt.windowSendQueue)),
		CurrentReceivedBps: float64(sum(rt.windowRecvQueue)),
		// 总量bps
		TotalSentBps:     float64(rt.totalSent.Load()) / secondTillNow,
		TotalReceivedBps: float64(rt.totalReceived.Load()) / secondTillNow,
	}
}

func (rt *Recorder) print() string {
	stats := rt.getStats()
	IP := fmt.Sprintf("[IP] %s, %s, %s", rt.ip, rt.startTime.Format("2006-01-02 15:04:05"), rt.lastHeartbeat.Format("15:04:05"))
	total := fmt.Sprintf("[Total] sent: %s, received: %s", formatBpsInt(stats.TotalSent), formatBpsInt(stats.TotalReceived))
	currentBps := fmt.Sprintf("[Current] sent: %s/s, received: %s/s", formatBps(stats.CurrentSentBps), formatBps(stats.CurrentReceivedBps))
	return fmt.Sprintf("%s\n%s\n%s\n", IP, total, currentBps)
}

func formatBps(b float64) string {
	if b < 1024 {
		return fmt.Sprintf("%8.3fB", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%8.3fKB", b/1024)
	}
	return fmt.Sprintf("%8.3fMB", b/1024/1024)
}

func formatBpsInt(b uint64) string {
	bf := float64(b)
	return formatBps(bf)
}

func sum(arr []uint64) uint64 {
	var s uint64
	for _, v := range arr {
		s += v
	}
	return s
}
