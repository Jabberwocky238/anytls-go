package main

import (
	F "anytls/addon/feedback"
	"anytls/util"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"flag"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

var passwordSha256 []byte

func main() {
	listen := flag.String("l", "0.0.0.0:9443", "redirect listen port")
	downstream := flag.String("s", "127.0.0.1:8443", "downstream anytls server")
	password := flag.String("p", "", "password")
	flag.Parse()

	if *password == "" {
		logrus.Fatalln("please set password")
	}

	logLevel, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		logLevel = logrus.DebugLevel
	}
	logrus.SetLevel(logLevel)

	var sum = sha256.Sum256([]byte(*password))
	passwordSha256 = sum[:]

	logrus.Infoln("[Redirect]", util.ProgramVersionName)
	logrus.Infoln("[Redirect] Listening TCP", *listen, "=>", *downstream)

	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		logrus.Fatalln("listen redirect tcp:", err)
	}

	// 生成自签 TLS 证书，和 server 端一致
	tlsCert, _ := util.GenerateKeyPair(time.Now, "")
	tlsConfigServer := &tls.Config{
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return tlsCert, nil
		},
	}
	// 下游客户端用的 tls.Config
	tlsConfigDownstream := &tls.Config{
		InsecureSkipVerify: true,
	}
	ctx, cancel := context.WithCancel(context.Background())

	// feedback
	_, port, err := net.SplitHostPort(*listen)
	if err != nil {
		logrus.Fatalln("split host port:", err)
	}
	portInt, err := strconv.Atoi(port)
	if err != nil {
		logrus.Fatalln("convert port:", err)
	}

	// 使用 myRedirector 封装
	redirector := NewMyRedirector(ctx, *downstream, tlsConfigDownstream)

	timer := F.NewTimer(*password, portInt, ctx, cancel)
	timer.Start()

	for {
		c, err := listener.Accept()
		if err != nil {
			logrus.Fatalln("accept:", err)
		}
		logrus.Infof("[Redirect] new client from %s", c.RemoteAddr())
		go handleClientConn(ctx, c, redirector, tlsConfigServer)

		select {
		case <-ctx.Done():
			timer.Stop()
			return
		default:
		}
	}
}
