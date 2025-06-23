package rate

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	windowSize        = 100 * time.Millisecond
	heartbeatDeadline = 1 * time.Minute
)

// RateTracker 流量跟踪器
type Recorder struct {
	// 总量统计
	totalSent     atomic.Uint64
	totalReceived atomic.Uint64
	startTime     time.Time
	ip            IP

	// 100ms窗口统计
	windowSent     atomic.Uint64
	windowReceived atomic.Uint64
	windowStart    time.Time

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
		startTime:   now,
		windowStart: now,
		ip:          ip,
		sendChan:    make(chan uint64, 1000), // 缓冲channel
		recvChan:    make(chan uint64, 1000), // 缓冲channel
		stopChan:    make(chan struct{}),
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

	ticker := time.NewTicker(windowSize)
	printer := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case sent := <-rt.sendChan:
			// 更新发送量
			rt.totalSent.Add(sent)
			rt.windowSent.Add(sent)

		case received := <-rt.recvChan:
			// 更新接收量
			rt.totalReceived.Add(received)
			rt.windowReceived.Add(received)

		case <-ticker.C:
			// 每100ms重置窗口
			rt.windowSent.Store(0)
			rt.windowReceived.Store(0)
			rt.windowStart = time.Now()

		case <-printer.C:
			logrus.Infof(rt.print())

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
	TotalSent          uint64 `json:"total_sent"`
	TotalReceived      uint64 `json:"total_received"`
	TotalSentBps       uint64 `json:"total_sent_bps"`
	TotalReceivedBps   uint64 `json:"total_received_bps"`
	CurrentSent        uint64 `json:"current_sent"`
	CurrentReceived    uint64 `json:"current_received"`
	CurrentSentBps     uint64 `json:"current_sent_bps"`
	CurrentReceivedBps uint64 `json:"current_received_bps"`
}

// GetStats 获取所有统计信息
func (rt *Recorder) getStats() Stats {
	windowTime := time.Since(rt.windowStart).Seconds()
	if windowTime <= 0 {
		windowTime = 0.1 // 避免除零
	}

	uptime := time.Since(rt.startTime).Seconds()
	if uptime <= 0 {
		uptime = 1 // 避免除零
	}

	return Stats{
		// 总量统计
		TotalSent:        rt.totalSent.Load(),
		TotalReceived:    rt.totalReceived.Load(),
		TotalSentBps:     uint64(float64(rt.totalSent.Load()*8) / uptime),
		TotalReceivedBps: uint64(float64(rt.totalReceived.Load()*8) / uptime),

		// 当前窗口统计
		CurrentSent:        rt.windowSent.Load(),
		CurrentReceived:    rt.windowReceived.Load(),
		CurrentSentBps:     uint64(float64(rt.windowSent.Load()*8) / windowTime),
		CurrentReceivedBps: uint64(float64(rt.windowReceived.Load()*8) / windowTime),
	}
}

func (rt *Recorder) print() string {
	stats := rt.getStats()
	IP := fmt.Sprintf("[IP] %s, %s", rt.ip, rt.startTime.Format("2006-01-02 15:04:05"))
	total := fmt.Sprintf("[Total] sent: %s, received: %s", formatBps(stats.TotalSent), formatBps(stats.TotalReceived))
	// totalBps := fmt.Sprintf("[Total Bps] sent: %d bps, received: %d bps", stats.TotalSentBps, stats.TotalReceivedBps)
	// current := fmt.Sprintf("[Current 100ms] sent: %d bytes, received: %d bytes", stats.CurrentSent, stats.CurrentReceived)
	currentBps := fmt.Sprintf("[Current] sent: %s bps, received: %s bps", formatBps(stats.CurrentSentBps), formatBps(stats.CurrentReceivedBps))
	return fmt.Sprintf("\n%s\n%s\n%s\n", IP, total, currentBps)
}

func formatBps(b uint64) string {
	if b < 1024 {
		return fmt.Sprintf("%dB", b/8)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%dKB", b/8/1024)
	}
	return fmt.Sprintf("%dMB", b/8/1024/1024)
}
