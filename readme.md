# AnyTLS

一个试图缓解 嵌套的TLS握手指纹(TLS in TLS) 问题的代理协议。`anytls-go` 是该协议的参考实现。

- 灵活的分包和填充策略
- 连接复用，降低代理延迟
- 简洁的配置

[用户常见问题](./docs/faq.md)

[协议文档](./docs/protocol.md)

[URI 格式](./docs/uri_scheme.md)

## 快速食用方法

### 服务器

```
anytls-server-windows.exe -l 127.0.0.1:8443 -p 1111qqqqjjjj
./anytls-server-linux -l 0.0.0.0:3306 -p 1111qqqqjjjj
ENV=prod ./anytls-server-linux -l 0.0.0.0:8080 -p 1111qqqqjjjjzq238_4
./anytls-server-linux-log -l 0.0.0.0:23877 -p 1111qqqqjjjjzq238_4
./anytls-redirect-linux -l 0.0.0.0:3306 -s 43.133.221.206:61555 -p 1111qqqqjjjjzq238_4
./anytls-redirect-linux -l 0.0.0.0:3306 -s 74.48.108.252:23877 -p 1111qqqqjjjjzq238_4

./anytls-redirect-linux -l 0.0.0.0:47073 -s 74.48.108.252:47073 -p zfEbVYzwh2YqZXWqgNftRVto
./anytls-redirect-linux -l 0.0.0.0:23892 -s 74.48.108.252:23822 -p 1111qqqqjjjjzq238_2
./anytls-redirect-linux -l 0.0.0.0:23893 -s 74.48.108.252:23833 -p 1111qqqqjjjjzq238_3
./anytls-redirect-linux -l 0.0.0.0:23894 -s 74.48.108.252:23844 -p 1111qqqqjjjjzq238_4
```

US美国洛杉矶-专线-1,121.5.40.207,23891,1111qqqqjjjjzq238_1,1
US美国洛杉矶-专线-2,121.5.40.207,23892,1111qqqqjjjjzq238_2,1
US美国洛杉矶-专线-3,121.5.40.207,23893,1111qqqqjjjjzq238_3,1
US美国洛杉矶-专线-4,121.5.40.207,23894,1111qqqqjjjjzq238_4,1

`0.0.0.0:8443` 为服务器监听的地址和端口。

### 客户端
tcpdump -i any -s 0 -w /tmp/redirect.pcap port 3306
```
./anytls-client-linux -l 0.0.0.0:3306 -s 43.133.221.206:61555 -p 1111qqqqjjjjzq238_4
curl --socks5 127.0.0.1:1081 http://www.gstatic.com/generate_204
curl --socks5 121.5.40.207:3306 http://www.gstatic.com/generate_204
anytls-client-windows.exe -l 127.0.0.1:1081 -s 121.5.40.207:3306 -p 1111qqqqjjjjzq238_4
anytls-client-windows.exe -l 127.0.0.1:1081 -s 43.133.221.206:61555 -p 1111qqqqjjjjzq238_4
anytls-client-windows.exe -l 127.0.0.1:1081 -s 74.48.108.252:23877 -p 1111qqqqjjjjzq238_4
```

`127.0.0.1:1080` 为本机 Socks5 代理监听地址，理论上支持 TCP 和 UDP(通过 udp over tcp 传输)。

### sing-box

https://github.com/SagerNet/sing-box

已合并至 dev-next 分支。它包含了 anytls 协议的服务器和客户端。

### mihomo

https://github.com/MetaCubeX/mihomo

已合并至 Alpha 分支。它包含了 anytls 协议的服务器和客户端。

### Shadowrocket

Shadowrocket 2.2.65+ 实现了 anytls 协议的客户端。
