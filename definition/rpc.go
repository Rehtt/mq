package definition

import "time"

const (
	MqRpcName = "MqRpc"

	AUTH             = MqRpcName + ".Auth"
	CREATE_MQ        = MqRpcName + ".CreateMq"
	DELETE_MQ        = MqRpcName + ".DeleteMq"
	PUSH             = MqRpcName + ".Push"
	READ             = MqRpcName + ".Read"
	POP              = MqRpcName + ".Pop"
	DELETE           = MqRpcName + ".Delete"
	DROP             = MqRpcName + ".Drop"
	ACTIVE           = MqRpcName + ".Active"
	PING             = MqRpcName + ".Ping"
	SET_KEY_VALUE    = MqRpcName + ".SetKeyValue"
	GET_KEY_VALUE    = MqRpcName + ".GetKeyValue"
	DELETE_KEY_VALUE = MqRpcName + ".DeleteKeyValue"
)

type MqRPC interface {
	// Auth(args AuthArgs, reply *AuthReply) (err error)

	CreateMq(args CreateMqArgs, reply *CreateMqReply) (err error)

	DeleteMq(args DeleteMqArgs, reply *DeleteMqReply) (err error)

	Push(args MqPushArgs, reply *MqPushReply) (err error)

	Read(args MqReadArgs, reply *MqReadReply) (err error)

	Pop(args MqPopArgs, reply *MqPopReply) (err error)

	Delete(args MqDeleteArgs, reply *MqDeleteReply) (err error)

	Drop(args MqDropArgs, reply *MqDropReply) (err error)

	Active(args MqActiveArgs, reply *MqActiveReply) (err error)

	SetKeyValue(args MqSetKeyValueArgs, reply *MqSetKeyValueReply) (err error)
	GetKeyValue(args MqGetKeyValueArgs, reply *MqGetKeyValueReply) (err error)
	DeleteKeyValue(args MqDeleteKeyValueArgs, reply *MqDeleteKeyValueReply) (err error)

	Ping(PingArgs, *PingReply) (err error)
}

// type AuthArgs struct {
// 	Token string
// }
// type AuthReply struct{}

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

type (
	PingArgs  struct{}
	PingReply struct{}
)

type MqSetKeyValueArgs struct {
	Mq     string
	Key    string
	Value  string
	Expire *time.Time
}
type MqSetKeyValueReply struct{}

type MqGetKeyValueArgs struct {
	Mq  string
	Key string
}
type MqGetKeyValueReply struct {
	Value *Value
	Ok    bool
}

type MqDeleteKeyValueArgs struct {
	Mq  string
	Key string
}
type MqDeleteKeyValueReply struct{}
