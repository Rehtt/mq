package sdk

import (
	"crypto/tls"
	"net/rpc"
	"net/url"
	"time"

	"github.com/Rehtt/mq/definition"
	"github.com/quic-go/quic-go"
	"golang.org/x/net/context"
)

var _ definition.Mq = (*MqClient)(nil)

type MqClient struct {
	client *rpc.Client
}

// 创建队列
func (m *MqClient) CreateMq(mq string) (err error) {
	return m.client.Call(definition.CREATE_MQ, definition.CreateMqArgs{Mq: mq}, nil)
}

// 删除队列
func (m *MqClient) DeleteMq(mq string) (err error) {
	return m.client.Call(definition.DELETE_MQ, definition.DeleteMqArgs{Mq: mq}, nil)
}

// 向队列里添加消息
func (m *MqClient) Push(mq string, msg string) (id uint64, err error) {
	var reply definition.MqPushReply
	err = m.client.Call(definition.PUSH, definition.MqPushArgs{Mq: mq, Msg: msg}, &reply)
	return reply.Id, err
}

// 读取指定条数消息，并设置超时时间
// 在时间范围内，消息不能再次被读取
// 如果超出时间范围没有将消息删除或归档，消息会在下次被读取
func (m *MqClient) Read(mq string, num int, timeout time.Duration) (msgs []definition.Msg, err error) {
	var reply definition.MqReadReply
	err = m.client.Call(definition.READ, definition.MqReadArgs{Mq: mq, Num: num, Timeout: timeout}, &reply)
	return reply.Msgs, err
}

// 从队列中读取指定条数消息
// 并在队列中删除消息
func (m *MqClient) Pop(mq string, num int) (msgs []definition.Msg, err error) {
	var reply definition.MqPopReply
	err = m.client.Call(definition.POP, definition.MqPopArgs{Mq: mq, Num: num}, &reply)
	return reply.Msgs, err
}

// 从队列中删除指定消息
func (m *MqClient) Delete(mq string, id uint64) (err error) {
	return m.client.Call(definition.DELETE, definition.MqDeleteArgs{Mq: mq, Id: id}, nil)
}

// 清空队列
func (m *MqClient) Drop(mq string) (err error) {
	return m.client.Call(definition.DROP, definition.MqDropArgs{Mq: mq}, nil)
}

// 将消息存档
func (m *MqClient) Active(mq string, id uint64) (err error) {
	return m.client.Call(definition.ACTIVE, definition.MqActiveArgs{Mq: mq, Id: id}, nil)
}

func (m *MqClient) Ping() (err error) {
	return m.client.Call(definition.PING, definition.PingArgs{}, nil)
}

func ConnectMq(ctx context.Context, addr string, safe bool) (*MqClient, error) {
	tlsConf := &tls.Config{
		NextProtos: []string{"mq"},
	}
	if !safe {
		tlsConf.InsecureSkipVerify = true
	}
	u, err := url.Parse(addr)
	if err == nil {
		tlsConf.ServerName = u.Host
	}

	quicClient, err := quic.DialAddr(ctx, addr, tlsConf, nil)
	if err != nil {
		return nil, err
	}
	stream, err := quicClient.OpenStream()
	if err != nil {
		return nil, err
	}
	client := rpc.NewClient(stream)
	mq := MqClient{client}
	go func() {
		t := time.NewTicker(time.Minute)
		for {
			<-t.C
			mq.Ping()
		}
	}()
	return &mq, nil
}
