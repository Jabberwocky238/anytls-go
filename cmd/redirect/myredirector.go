package main

import (
	"anytls/proxy"
	"anytls/proxy/session"
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"

	M "github.com/sagernet/sing/common/metadata"
)

// DialFunc 定义出站拨号函数类型
// 可自定义实现（如 anytls、socks5、tls 等）
type DialFunc func(ctx context.Context, addr net.Addr) (net.Conn, error)

// Redirection 表示一次转发请求
// InboundConn: 入站连接
// TargetAddr: 目标地址
// Dial: 出站拨号函数
type Redirection struct {
	InboundConn net.Conn
	TargetAddr  net.Addr
	Dial        DialFunc
}

// Redirector 异步转发调度器
// 支持高并发安全投递
type Redirector struct {
	ctx             context.Context
	redirectionChan chan *Redirection
}

// NewRedirector 创建新的 Redirector
func NewRedirector(ctx context.Context) *Redirector {
	r := &Redirector{
		ctx:             ctx,
		redirectionChan: make(chan *Redirection, 64),
	}
	go r.worker()
	return r
}

// Redirect 投递转发请求
func (r *Redirector) Redirect(redir *Redirection) {
	select {
	case r.redirectionChan <- redir:
		// 投递成功
	case <-r.ctx.Done():
		// 退出
	}
}

// worker 后台协程，异步处理转发
func (r *Redirector) worker() {
	for {
		select {
		case redir := <-r.redirectionChan:
			go r.handle(redir)
		case <-r.ctx.Done():
			return
		}
	}
}

// handle 实际处理单次转发
func (r *Redirector) handle(redir *Redirection) {
	if redir.InboundConn == nil || redir.TargetAddr == nil || redir.Dial == nil {
		return
	}
	defer redir.InboundConn.Close()
	ctx := r.ctx
	outConn, err := redir.Dial(ctx, redir.TargetAddr)
	if err != nil {
		return
	}
	defer outConn.Close()
	errChan := make(chan error, 2)
	copyConn := func(dst, src net.Conn) {
		_, err := io.Copy(dst, src)
		errChan <- err
	}
	go copyConn(outConn, redir.InboundConn)
	go copyConn(redir.InboundConn, outConn)
	select {
	case <-errChan:
		// 一方断开即结束
	case <-ctx.Done():
	}
}

type myRedirector struct {
	client *session.Client
}

// NewMyRedirector 初始化 myRedirector，内部维护 session pool 到下游 server
func NewMyRedirector(ctx context.Context, downstream string, tlsConfig *tls.Config) *myRedirector {
	client := session.NewClient(ctx, func(ctx context.Context) (net.Conn, error) {
		conn, err := proxy.SystemDialer.DialContext(ctx, "tcp", downstream)
		if err != nil {
			return nil, err
		}
		conn = tls.Client(conn, tlsConfig)
		// 写入密码和padding
		b := make([]byte, 34)
		copy(b[:32], passwordSha256)
		b[32] = 0
		b[33] = 0 // 无padding
		_, err = conn.Write(b)
		if err != nil {
			conn.Close()
			return nil, err
		}
		return conn, nil
	}, nil, 5*time.Second, 5*time.Second, 4)
	return &myRedirector{client: client}
}

// CreateProxy 创建到下游 server 的 stream，返回 net.Conn（协议式转发，写入目标地址到下游server）
func (r *myRedirector) CreateProxy(ctx context.Context, targetAddr net.Addr) (net.Conn, error) {
	stream, err := r.client.CreateStream(ctx)
	if err != nil {
		return nil, err
	}
	// 直接断言为 M.Socksaddr 类型，确保协议一致
	sa := targetAddr.(M.Socksaddr)
	if err := M.SocksaddrSerializer.WriteAddrPort(stream, sa); err != nil {
		stream.Close()
		return nil, err
	}
	return stream, nil
}
