## 轻量级 MQ（Go + gRPC/HTTP2）

一个使用 Go 实现的轻量级消息队列，支持 TLS 与简单认证，提供队列与 KV 能力，适合在本地或容器中快速部署与使用。

### 功能
- [x] 创建/删除/清空队列：`CreateMq` / `DeleteMq` / `Drop`
- [x] 写入/读取/弹出消息：`Push` / `Read`（带超时隐藏）/ `Pop`
- [x] 删除/归档消息：`Delete` / `Active`
- [x] 队列长度：`Len`
- [x] KV 能力：`SetKeyValue` / `GetKeyValue` / `DeleteKeyValue`

### 安装与构建
- 直接安装可执行文件
```bash
go install github.com/Rehtt/mq@latest
```

- 生成 gRPC 代码（如修改了 `api/proto/mq.proto`）
```bash
make genapi
```

- 构建二进制
```bash
make build
# 产物位于 ./bin/mq
```

### 运行服务
首次运行会在工作目录自动生成 TLS 证书（`cert.pem`/`key.pem`）。

```bash
mkdir -p ~/.config/mq

# 以默认配置运行（gRPC+TLS）
./bin/mq \
  -addr :1234 \
  -path ~/.config/mq \
  -cert cert.pem \
  -key  key.pem \
  -password "your_password"
```

参数说明：
- `-addr`：服务监听地址，默认`:1234`
- `-path`：工作目录（数据库、证书默认放置在此）
- `-cert`/`-key`：TLS 证书/私钥路径（如不存在会自动生成）
- `-password`：简单认证口令（可空）

### 认证
客户端通过 gRPC metadata 发送 Authorization 头：
- Header：`authorization: Bearer @your_password@`

### Go SDK 使用示例
```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/Rehtt/mq/sdk"
)

func main() {
    ctx := context.Background()
    client, err := sdk.ConnectMq(ctx, "127.0.0.1:1234", false, "your_password")
    if err != nil { log.Fatal(err) }
    defer client.Close()

    mq := "demo"
    _ = client.DeleteMq(mq)
    if err := client.CreateMq(mq); err != nil { log.Fatal(err) }

    id, _ := client.Push(mq, "hello")
    msgs, _ := client.Read(mq, 1, 2*time.Second)
    _ = client.Active(mq, id)
    l, _ := client.Len(mq)
    _ = client.SetKeyValue(mq, "k", "v", 5*time.Second)
    v, ok, _ := client.GetKeyValue(mq, "k")
    _ = client.DeleteMq(mq)

    log.Println(id, len(msgs), l, ok, v != nil)
}
```

### API 定义
- Proto：`api/proto/mq.proto`
- 生成：`make genapi`（会生成到 `api/pb/`）

### 测试与基准
- 运行核心与服务端测试：
```bash
go test -v ./internal/... ./server/ 
```

- SDK 测试需要服务已启动（否则会跳过/失败）：
```bash
go test -v ./sdk
```

- 运行性能基准（部分用例）：
```bash
go test -bench=. -benchmem ./internal/mq -run Benchmark -timeout 30s
```

### Docker 运行
```bash
docker run -d \
  -p 1234:1234 \
  -v $(pwd)/data:/data \
  ghcr.io/rehtt/mq:latest \
  -addr :1234 -path /data -password "your_password"
```

### 目录结构（简要）
```
.
├── api/                 # proto 与生成代码
├── internal/
│   ├── errors/          # 统一错误类型与工具
│   └── mq/              # 核心业务（内存结构、仓库实现、测试、基准）
├── server/              # gRPC 服务器、认证与测试
├── sdk/                 # Go 客户端 SDK
├── main.go              # 入口
└── tls.go               # TLS 证书生成
```

### 重要变更（迁移说明）
- 已移除旧的 RPC + QUIC 实现，统一为 gRPC + HTTP/2
- 保留原有业务语义与能力，接口通过 protobuf 定义

### 许可证
MIT
