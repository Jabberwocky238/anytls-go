package feedback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Timer 定时器结构
type Timer struct {
	password   string
	port       int
	host       string
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

var ServerURL string

var (
	RegisterRetryInterval = 15 * time.Second // 注册重试初始间隔
	RegisterRetryMax      = 5                // 注册最大重试次数
	HeartbeatInterval     = 5 * time.Second  // 心跳间隔
	DefaultInterval       = 15 * time.Second // 默认心跳间隔
	HTTPTimeout           = 10 * time.Second // HTTP超时时间
)

func init() {
	if os.Getenv("ENV") == "prod" {
		if err := godotenv.Load(".env"); err != nil {
			fmt.Printf("无法加载.env文件: %v", err)
		}
	} else {
		if err := godotenv.Load("../.env.dev"); err != nil {
			fmt.Printf("无法加载.env文件: %v", err)
		}
	}
	ServerURL = os.Getenv("API_BASE_URL") + "/proxy"
}

// NewTimer 创建新的定时器
func NewTimer(password string, port int, ctx context.Context, cancel context.CancelFunc) *Timer {
	ip, err := GetPublicIP()
	if err != nil {
		logrus.Fatalln("get public ip:", err)
	}
	logrus.Debugf("public endpoint: %s:%d", ip, port)

	return &Timer{
		password: password,
		port:     port,
		host:     ip,
		httpClient: &http.Client{
			Timeout: HTTPTimeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动定时器
func (t *Timer) Start() error {
	if err := t.retryRegister(); err != nil {
		return err
	}

	go t.heartbeatLoop()
	return nil
}

// Stop 停止定时器
func (t *Timer) Stop() {
	t.cancel()
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

// sendInitialRequest 发送初始请求
func (t *Timer) sendInitialRequest() error {
	req := ProxyRegisterRequest{
		Host:     t.host,
		Port:     t.port,
		Password: t.password,
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
		Host: t.host,
		Port: t.port,
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

// retryRegister 封装注册重试逻辑，失败间隔x2，最多max次，成功返回nil，失败返回最后一次错误
func (t *Timer) retryRegister() error {
	failCount := 0
	interval := RegisterRetryInterval
	for {
		err := t.sendInitialRequest()
		if err == nil {
			logrus.Debugf("注册成功")
			return nil
		}
		failCount++
		if failCount >= RegisterRetryMax {
			return fmt.Errorf("注册失败超过%d次，退出: %v", RegisterRetryMax, err)
		}
		logrus.Debugf("注册失败: %v，%d秒后重试（第%d次）...", err, int(interval.Seconds()), failCount)
		if !t.waitOrCancel(interval) {
			return fmt.Errorf("启动被取消")
		}
		interval *= 2
	}
}

// waitOrCancel 等待指定时间或ctx取消，返回true表示正常等待，false表示被取消
func (t *Timer) waitOrCancel(d time.Duration) bool {
	select {
	case <-t.ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// heartbeatLoop 心跳循环
func (t *Timer) heartbeatLoop() {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			if err := t.sendHeartbeat(); err != nil {
				fmt.Printf("发送心跳失败: %v\n", err)
				ticker.Stop()
				if err := t.retryRegister(); err != nil {
					fmt.Printf("重新注册失败，退出心跳: %v\n", err)
					t.cancel()
					return
				}
				// 注册成功后递归重启心跳
				go t.heartbeatLoop()
				return
			}
		}
	}
}
