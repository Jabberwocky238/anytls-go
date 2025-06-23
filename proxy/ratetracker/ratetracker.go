package ratetracker

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	windowSize = 100 * time.Millisecond
)

// RateTracker 流量跟踪器
type RateTracker struct {
	// 总量统计
	totalSent     atomic.Uint64
	totalReceived atomic.Uint64
	startTime     time.Time
	ip            string

	// 100ms窗口统计
	windowSent     atomic.Uint64
	windowReceived atomic.Uint64
	windowStart    time.Time

	// channel驱动
	sendChan chan uint64
	recvChan chan uint64
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewRateTracker 创建新的跟踪器
func NewRateTracker(ip string) *RateTracker {
	now := time.Now()
	rt := &RateTracker{
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
func (rt *RateTracker) SendChan() chan<- uint64 {
	return rt.sendChan
}

// RecvChan 获取接收channel
func (rt *RateTracker) RecvChan() chan<- uint64 {
	return rt.recvChan
}

// recordLoop 自动记录循环
func (rt *RateTracker) recordLoop() {
	defer rt.wg.Done()

	ticker := time.NewTicker(windowSize)
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

		case <-rt.stopChan:
			return
		}
	}
}

// Stop 停止跟踪器
func (rt *RateTracker) Stop() {
	close(rt.stopChan)
	rt.wg.Wait()
}

// GetStats 获取所有统计信息
func (rt *RateTracker) GetStats() map[string]uint64 {
	windowTime := time.Since(rt.windowStart).Seconds()
	if windowTime <= 0 {
		windowTime = 0.1 // 避免除零
	}

	uptime := time.Since(rt.startTime).Seconds()
	if uptime <= 0 {
		uptime = 1 // 避免除零
	}

	return map[string]uint64{
		// 总量统计
		"total_sent":         rt.totalSent.Load(),
		"total_received":     rt.totalReceived.Load(),
		"total_sent_bps":     uint64(float64(rt.totalSent.Load()*8) / uptime),
		"total_received_bps": uint64(float64(rt.totalReceived.Load()*8) / uptime),

		// 当前窗口统计
		"current_sent":         rt.windowSent.Load(),
		"current_received":     rt.windowReceived.Load(),
		"current_sent_bps":     uint64(float64(rt.windowSent.Load()*8) / windowTime),
		"current_received_bps": uint64(float64(rt.windowReceived.Load()*8) / windowTime),
	}
}

func (rt *RateTracker) Print() string {
	stats := rt.GetStats()
	IP := fmt.Sprintf("[IP] %s, %s", rt.ip, rt.startTime.Format("2006-01-02 15:04:05"))
	total := fmt.Sprintf("[Total] sent: %d bytes, received: %d bytes", stats["total_sent"], stats["total_received"])
	totalBps := fmt.Sprintf("[Total Bps] sent: %d bps, received: %d bps", stats["total_sent_bps"], stats["total_received_bps"])
	current := fmt.Sprintf("[Current 100ms] sent: %d bytes, received: %d bytes", stats["current_sent"], stats["current_received"])
	currentBps := fmt.Sprintf("[Current Bps] sent: %d bps, received: %d bps", stats["current_sent_bps"], stats["current_received_bps"])
	return fmt.Sprintf("\n%s\n%s\n%s\n%s\n%s\n", IP, total, totalBps, current, currentBps)
}
