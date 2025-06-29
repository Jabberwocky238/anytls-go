package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"net"
	"runtime/debug"
	"time"

	"anytls/proxy/padding"
	"anytls/proxy/session"

	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sirupsen/logrus"
)

// handleClientConn 处理每个 client 连接，认证、解包，复用 session pool 创建 stream，转发流量
func handleClientConn(ctx context.Context, c net.Conn, myRedirector *myRedirector, tlsConfig *tls.Config) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorln("[BUG]", r, string(debug.Stack()))
		}
		logrus.Debugf("[Redirect] client %s: handler exit", c.RemoteAddr())
	}()

	c.SetDeadline(time.Now().Add(10 * time.Second))
	c = tls.Server(c, tlsConfig)
	defer func() {
		logrus.Debugf("[Redirect] client %s: connection closed", c.RemoteAddr())
		c.Close()
	}()

	b := buf.NewPacket()
	defer func() {
		logrus.Debugf("[Redirect] client %s: buffer released", c.RemoteAddr())
		b.Release()
	}()

	// n, err := b.ReadOnceFrom(c)
	_, err := b.ReadOnceFrom(c)
	if err != nil {
		logrus.Warnf("[Redirect] ReadOnceFrom %s failed: %v", c.RemoteAddr(), err)
		return
	}
	c = bufio.NewCachedConn(c, b)

	by, err := b.ReadBytes(32)
	if err != nil || !bytes.Equal(by, passwordSha256) {
		logrus.Warnf("[Redirect] client %s auth failed, got: %x, want: %x", c.RemoteAddr(), by, passwordSha256)
		if err == nil && len(by) == 32 {
			peek, _ := b.ReadBytes(2)
			logrus.Warnf("[Redirect] client %s next 2 bytes (paddingLen): %x", c.RemoteAddr(), peek)
		}
		return
	}

	by, err = b.ReadBytes(2)
	if err != nil {
		logrus.Warnf("[Redirect] client %s read padding failed: %v", c.RemoteAddr(), err)
		return
	}
	paddingLen := binary.BigEndian.Uint16(by)
	if paddingLen > 0 {
		_, err = b.ReadBytes(int(paddingLen))
		if err != nil {
			logrus.Warnf("[Redirect] client %s read padding bytes failed: %v", c.RemoteAddr(), err)
			return
		}
	}

	// 正确处理 anytls 协议，参考 server 端实现
	logrus.Debugf("[Redirect] start session for %s", c.RemoteAddr())
	session := session.NewServerSession(c, func(stream *session.Stream) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorln("[BUG]", r, string(debug.Stack()))
			}
		}()
		defer func() {
			logrus.Debugf("[Redirect] stream closed for %s", c.RemoteAddr())
			stream.Close()
		}()

		logrus.Debugf("[Redirect] waiting for destination from %s", c.RemoteAddr())
		destination, err := M.SocksaddrSerializer.ReadAddrPort(stream)
		if err != nil {
			logrus.Debugf("[Redirect] ReadAddrPort failed for %s: %v", c.RemoteAddr(), err)
			return
		}
		logrus.Infof("[Redirect] 收到目标地址类型: %T, 值: %s", destination, destination.String())
		logrus.Infof("[Redirect] got destination for %s: %s", c.RemoteAddr(), destination.String())

		// 创建到下游 server 的代理连接
		proxyStream, err := myRedirector.CreateProxy(ctx, destination)
		if err != nil {
			logrus.Errorf("[Redirect] create proxy for %s failed: %v", c.RemoteAddr(), err)
			return
		}
		defer proxyStream.Close()

		logrus.Infof("[Redirect] start relay %s <-> %s", c.RemoteAddr(), destination.String())
		done := make(chan struct{}, 2)
		go func() {
			err := bufio.CopyConn(ctx, proxyStream, stream)
			if err != nil {
				logrus.Warnf("[Redirect] relay downstream->client error for %s: %v", c.RemoteAddr(), err)
			}
			done <- struct{}{}
		}()
		go func() {
			err := bufio.CopyConn(ctx, stream, proxyStream)
			if err != nil {
				logrus.Warnf("[Redirect] relay client->downstream error for %s: %v", c.RemoteAddr(), err)
			}
			done <- struct{}{}
		}()
		<-done
		logrus.Infof("[Redirect] relay finished for %s", c.RemoteAddr())
	}, &padding.DefaultPaddingFactory)
	session.Run()
	session.Close()
	logrus.Debugf("[Redirect] session closed for %s", c.RemoteAddr())
}

// 新增辅助函数
func parseDomainFromSocksaddr(addr net.Addr) (string, uint16, bool) {
	type domainPort interface {
		Domain() string
		Port() uint16
	}
	if d, ok := addr.(domainPort); ok {
		return d.Domain(), d.Port(), true
	}
	return "", 0, false
}
