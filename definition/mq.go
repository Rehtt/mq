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

const (
	MqRpcName = "MqRpc"

	CREATE_MQ = MqRpcName + ".CreateMq"
	DELETE_MQ = MqRpcName + ".DeleteMq"
	PUSH      = MqRpcName + ".Push"
	READ      = MqRpcName + ".Read"
	POP       = MqRpcName + ".Pop"
	DELETE    = MqRpcName + ".Delete"
	DROP      = MqRpcName + ".Drop"
	ACTIVE    = MqRpcName + ".Active"
)

type MqRPC interface {
	CreateMq(args CreateMqArgs, reply *CreateMqReply) (err error)

	DeleteMq(args DeleteMqArgs, reply *DeleteMqReply) (err error)

	Push(args MqPushArgs, reply *MqPushReply) (err error)

	Read(args MqReadArgs, reply *MqReadReply) (err error)

	Pop(args MqPopArgs, reply *MqPopReply) (err error)

	Delete(args MqDeleteArgs, reply *MqDeleteReply) (err error)

	Drop(args MqDropArgs, reply *MqDropReply) (err error)

	Active(args MqActiveArgs, reply *MqActiveReply) (err error)
}

type CreateMqArgs struct {
	Mq string
}
type CreateMqReply struct{}

type DeleteMqArgs struct {
	Mq string
}
type DeleteMqReply struct{}

type MqPushArgs struct {
	Mq  string
	Msg string
}
type MqPushReply struct {
	Id uint64
}

type MqReadArgs struct {
	Mq      string
	Num     int
	Timeout time.Duration
}
type MqReadReply struct {
	Msgs []Msg
}

type MqPopArgs struct {
	Mq  string
	Num int
}
type MqPopReply struct {
	Msgs []Msg
}

type MqDeleteArgs struct {
	Mq string
	Id uint64
}
type MqDeleteReply struct{}

type MqDropArgs struct {
	Mq string
}
type MqDropReply struct{}

type MqActiveArgs struct {
	Mq string
	Id uint64
}
type MqActiveReply struct{}
