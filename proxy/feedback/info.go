package feedback

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// GetPublicIP 获取当前主机的公网IP地址
func GetPublicIP() (string, error) {
	resp, err := http.Get("https://ifconfig.me/ip")
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
	resp, err := http.Get("http://ip-api.com/line/" + ip + "?fields=country")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	country := strings.TrimSpace(string(body))
	if country == "" || country == "fail" {
		return "", errors.New("无法获取国家信息")
	}
	return country, nil
}
