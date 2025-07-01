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
