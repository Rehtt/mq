## 使用golang实现的轻量级mq

### 支持功能
- [x] 创建队列 `CreateMq(mq string)`
- [x] 删除队列 `DeleteMq(mq string)`
- [x] 清空队列 `Drop(mq string)`
- [x] push消息 `Push(mq string, msg string)`
- [x] pop消息 `Pop(mq string, num int)`
- [x] 归档消息 `Archive(mq string, id uint64)`
- [x] 读取消息并设置时间等待归档或删除，等待时间内该消息不会再被读取 `Read(mq string, num int, timeout time.Duration)`
- [x] 删除消息 `Delete(mq string, id uint64)`

### 使用

#### golang

```sh
go install github.com/Rehtt/mq@latest

mkdir ~/.config/mq
mq -path ~/.config/mq/
```

#### docker
```sh
docker run -d -p 1234:1234 -v data:/data ghcr.io/rehtt/mq:master
```
