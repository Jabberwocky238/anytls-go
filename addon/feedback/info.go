package feedback

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func getIPv4Client() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
		// 这里可以加更多 dialer 配置
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr) // 强制用tcp4
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
}

var client = getIPv4Client()

// GetPublicIP 获取当前主机的公网IP地址
func GetPublicIP() (string, error) {
	resp, err := client.Get("https://ifconfig.me/ip")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	ipstr := strings.TrimSpace(string(body))
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return "", errors.New("invalid ip")
	}
	if ip.To4() != nil {
		return ip.String(), nil
	}
	if ip.To16() != nil {
		return fmt.Sprintf("[%s]", ip.String()), nil
	}
	return "", errors.New("invalid ip")
}

// GetIPCountry 获取指定IP的国家信息
func GetIPCountry(ip string) (string, error) {
	logrus.Println("ip", ip)
	resp, err := client.Get("http://ip-api.com/line/" + ip + "?fields=country")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	logrus.Println("body", string(body))
	country := strings.TrimSpace(string(body))
	logrus.Println("country", country)
	if country == "" || country == "fail" {
		return "", errors.New("无法获取国家信息")
	}
	return country, nil
}
