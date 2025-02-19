package definition

import (
	"time"
)

type Msg struct {
	Id        uint64
	Text      string
	CreatedAt time.Time
}

type Mq interface {
	// 创建队列
	CreateMq(mq string) (err error)

	// 删除队列
	DeleteMq(mq string) (err error)

	// 向队列里添加消息
	Push(mq string, msg string) (id uint64, err error)

	// 读取指定条数消息，并设置超时时间
	// 在时间范围内，消息不能再次被读取
	// 如果超出时间范围没有将消息删除或归档，消息会在下次被读取
	Read(mq string, num int, timeout time.Duration) (msgs []Msg, err error)

	// 从队列中读取指定条数消息
	// 并在队列中删除消息
	Pop(mq string, num int) (msgs []Msg, err error)

	// 从队列中删除指定消息
	Delete(mq string, id uint64) (err error)

	// 清空队列
	Drop(mq string) (err error)

	// 将消息存档
	Active(mq string, id uint64) (err error)
}
