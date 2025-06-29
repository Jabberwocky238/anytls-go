package main

import (
	"anytls/proxy/padding"
	"anytls/proxy/session"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"net"
	"runtime/debug"
	"strings"

	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sirupsen/logrus"
)

func handleTcpConnection(ctx context.Context, c net.Conn, s *myServer) {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorln("[BUG]", r, string(debug.Stack()))
		}
	}()

	logrus.Debugf("[Server] new connection from %s", c.RemoteAddr())
	c = tls.Server(c, s.tlsConfig)
	defer func() {
		logrus.Debugf("[Server] connection from %s closed", c.RemoteAddr())
		c.Close()
	}()

	b := buf.NewPacket()
	defer func() {
		logrus.Debugf("[Server] buffer released for %s", c.RemoteAddr())
		b.Release()
	}()

	n, err := b.ReadOnceFrom(c)
	if err != nil {
		logrus.Debugf("[Server] ReadOnceFrom %s failed: %v", c.RemoteAddr(), err)
		return
	}
	logrus.Debugf("[Server] ReadOnceFrom %s success, n=%d", c.RemoteAddr(), n)
	c = bufio.NewCachedConn(c, b)

	by, err := b.ReadBytes(32)
	if err != nil || !bytes.Equal(by, passwordSha256) {
		logrus.Debugf("[Server] auth failed for %s, got: %x, want: %x", c.RemoteAddr(), by, passwordSha256)
		b.Resize(0, n)
		fallback(ctx, c)
		return
	}
	logrus.Debugf("[Server] auth success for %s", c.RemoteAddr())
	by, err = b.ReadBytes(2)
	if err != nil {
		logrus.Debugf("[Server] read padding failed for %s: %v", c.RemoteAddr(), err)
		b.Resize(0, n)
		fallback(ctx, c)
		return
	}
	paddingLen := binary.BigEndian.Uint16(by)
	logrus.Debugf("[Server] paddingLen for %s: %d", c.RemoteAddr(), paddingLen)
	if paddingLen > 0 {
		_, err = b.ReadBytes(int(paddingLen))
		if err != nil {
			logrus.Debugf("[Server] read padding bytes failed for %s: %v", c.RemoteAddr(), err)
			b.Resize(0, n)
			fallback(ctx, c)
			return
		}
	}

	logrus.Debugf("[Server] start session for %s", c.RemoteAddr())
	session := session.NewServerSession(c, func(stream *session.Stream) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorln("[BUG]", r, string(debug.Stack()))
			}
		}()
		defer func() {
			logrus.Debugf("[Server] stream closed for %s", c.RemoteAddr())
			stream.Close()
		}()

		logrus.Debugf("[Server] waiting for destination from %s", c.RemoteAddr())
		destination, err := M.SocksaddrSerializer.ReadAddrPort(stream)
		if err != nil {
			logrus.Debugf("[Server] ReadAddrPort failed for %s: %v", c.RemoteAddr(), err)
			return
		}
		logrus.Debugf("[Server] got destination for %s: %s", c.RemoteAddr(), destination.String())

		if strings.Contains(destination.String(), "udp-over-tcp.arpa") {
			logrus.Debugf("[Server] proxyOutboundUoT for %s", c.RemoteAddr())
			proxyOutboundUoT(ctx, stream, destination)
		} else {
			logrus.Debugf("[Server] proxyOutboundTCP for %s", c.RemoteAddr())
			proxyOutboundTCP(ctx, stream, destination)
		}
	}, &padding.DefaultPaddingFactory)
	session.Run()
	session.Close()
	logrus.Debugf("[Server] session closed for %s", c.RemoteAddr())
}

func fallback(ctx context.Context, c net.Conn) {
	// 暂未实现
	logrus.Debugln("fallback:", c.RemoteAddr())
}
