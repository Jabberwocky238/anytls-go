package main

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/sagernet/sing/common/buf"
)

// AnyTLSDial 作为出站拨号函数，复用 client 逻辑，连接下游 anytls server
func AnyTLSDial(ctx context.Context, addr net.Addr) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		return nil, err
	}
	conn = tls.Client(conn, &tls.Config{InsecureSkipVerify: true})

	b := buf.NewPacket()
	defer b.Release()
	b.Write(passwordSha256)
	b.Write([]byte{0, 0}) // 无 padding
	_, err = b.WriteTo(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	// 这里目标地址会在 Redirection 里写入
	return conn, nil
}
