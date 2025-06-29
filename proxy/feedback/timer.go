package feedback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Config 配置结构
type Config struct {
	Password string        // 密码
	Host     string        // 主机
	Port     int           // 端口
	Interval time.Duration // 心跳间隔，默认15秒
}

const (
	ServerURL = "http://127.0.0.1:8877/proxy"
)

// Timer 定时器结构
type Timer struct {
	config     *Config
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

// ProxyRegisterRequest 代理注册请求
type ProxyRegisterRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port" binding:"required"`
	Password string `json:"password" binding:"required"`
	VIP      bool   `json:"vip"`
	Country  string `json:"country" binding:"required"`
}

// ProxyHeartbeatRequest 代理心跳请求
type ProxyHeartbeatRequest struct {
	Host string `json:"host" binding:"required"`
	Port int    `json:"port" binding:"required"`
	Time int64  `json:"time"`
}

// NewTimer 创建新的定时器
func NewTimer(config *Config) *Timer {
	if config.Interval == 0 {
		config.Interval = 15 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Timer{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动定时器
func (t *Timer) Start() error {
	// 启动时发送初始请求
	if err := t.sendInitialRequest(); err != nil {
		return fmt.Errorf("发送初始请求失败: %w", err)
	}

	// 启动心跳定时器
	go t.heartbeatLoop()

	return nil
}

// Stop 停止定时器
func (t *Timer) Stop() {
	t.cancel()
}

// sendInitialRequest 发送初始请求
func (t *Timer) sendInitialRequest() error {
	req := ProxyRegisterRequest{
		Host:     t.config.Host,
		Port:     t.config.Port,
		Password: t.config.Password,
		VIP:      false,
		Country:  "CN",
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}
	buffer := bytes.NewBuffer(jsonData)
	return t.sendRequest(buffer, ServerURL+"/register")
}

// sendHeartbeat 发送心跳请求
func (t *Timer) sendHeartbeat() error {
	req := ProxyHeartbeatRequest{
		Host: t.config.Host,
		Port: t.config.Port,
		Time: time.Now().Unix(),
	}
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}
	buffer := bytes.NewBuffer(jsonData)
	return t.sendRequest(buffer, ServerURL+"/heartbeat")
}

// sendRequest 发送HTTP请求
func (t *Timer) sendRequest(req *bytes.Buffer, url string) error {
	httpReq, err := http.NewRequestWithContext(t.ctx, "POST", url, req)
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// heartbeatLoop 心跳循环
func (t *Timer) heartbeatLoop() {
	ticker := time.NewTicker(t.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if err := t.sendHeartbeat(); err != nil {
				fmt.Printf("发送心跳失败: %v\n", err)
			}
		}
	}
}
