package main

import (
	"time"

	"github.com/Rehtt/mq/definition"
)

type MqRpc struct {
	mq *Mq
}

var _ definition.MqRPC = (*MqRpc)(nil)

func (m *MqRpc) Ping(_ definition.PingArgs, _ *definition.PingReply) (err error) {
	return nil
}

func (m *MqRpc) CreateMq(args definition.CreateMqArgs, reply *definition.CreateMqReply) (err error) {
	return m.mq.CreateMq(args.Mq)
}

func (m *MqRpc) DeleteMq(args definition.DeleteMqArgs, reply *definition.DeleteMqReply) (err error) {
	return m.mq.DeleteMq(args.Mq)
}

func (m *MqRpc) Push(args definition.MqPushArgs, reply *definition.MqPushReply) (err error) {
	id, err := m.mq.Push(args.Mq, args.Msg)
	reply.Id = id
	return err
}

func (m *MqRpc) Read(args definition.MqReadArgs, reply *definition.MqReadReply) (err error) {
	msgs, err := m.mq.Read(args.Mq, args.Num, args.Timeout)
	reply.Msgs = msgs
	return err
}

func (m *MqRpc) Pop(args definition.MqPopArgs, reply *definition.MqPopReply) (err error) {
	msgs, err := m.mq.Pop(args.Mq, args.Num)
	reply.Msgs = msgs
	return err
}

func (m *MqRpc) Delete(args definition.MqDeleteArgs, reply *definition.MqDeleteReply) (err error) {
	return m.mq.Delete(args.Mq, args.Id)
}

func (m *MqRpc) Drop(args definition.MqDropArgs, reply *definition.MqDropReply) (err error) {
	return m.mq.Drop(args.Mq)
}

func (m *MqRpc) Active(args definition.MqActiveArgs, reply *definition.MqActiveReply) (err error) {
	return m.mq.Active(args.Mq, args.Id)
}

func (m *MqRpc) SetKeyValue(args definition.MqSetKeyValueArgs, reply *definition.MqSetKeyValueReply) (err error) {
	var exp time.Duration
	if args.Expire != nil && !args.Expire.IsZero() {
		exp = args.Expire.Sub(time.Now())
		if exp < 0 {
			return nil
		}
	}
	return m.mq.SetKeyValue(args.Mq, args.Key, args.Value, exp)
}

func (m *MqRpc) GetKeyValue(args definition.MqGetKeyValueArgs, reply *definition.MqGetKeyValueReply) (err error) {
	reply.Value, reply.Ok, err = m.mq.GetKeyValue(args.Mq, args.Key)
	return
}

func (m *MqRpc) DeleteKeyValue(args definition.MqDeleteKeyValueArgs, reply *definition.MqDeleteKeyValueReply) (err error) {
	return m.mq.DeleteKeyValue(args.Mq, args.Key)
}

func (m *MqRpc) Len(args definition.MqLenArgs, reply *definition.MqLenReply) (err error) {
	reply.Len, err = m.mq.Len(args.Mq)
	return err
}

func NewMqRpc() *MqRpc {
	return &MqRpc{
		mq: NewMq(),
	}
}
