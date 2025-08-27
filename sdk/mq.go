package sdk

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/Rehtt/mq/api/pb"
	"github.com/Rehtt/mq/definition"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var _ definition.Mq = (*MqClient)(nil)

// MqClient gRPC客户端
type MqClient struct {
	client pb.MQClient
	conn   *grpc.ClientConn
	ctx    context.Context
}

// 创建队列
func (m *MqClient) CreateMq(mq string) (err error) {
	req := &pb.CreateMqRequest{Mq: mq}
	resp, err := m.client.CreateMq(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

// 删除队列
func (m *MqClient) DeleteMq(mq string) (err error) {
	req := &pb.DeleteMqRequest{Mq: mq}
	resp, err := m.client.DeleteMq(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

// 向队列里添加消息
func (m *MqClient) Push(mq string, msg string) (id uint64, err error) {
	req := &pb.PushRequest{Mq: mq, Msg: msg}
	resp, err := m.client.Push(m.ctx, req)
	if err != nil {
		return 0, err
	}
	if resp.Error != "" {
		return 0, &RpcError{Message: resp.Error}
	}
	return resp.Id, nil
}

// 读取指定条数消息，并设置超时时间
func (m *MqClient) Read(mq string, num int, timeout time.Duration) (msgs []definition.Msg, err error) {
	req := &pb.ReadRequest{
		Mq:      mq,
		Num:     int32(num),
		Timeout: int64(timeout.Seconds()),
	}
	resp, err := m.client.Read(m.ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, &RpcError{Message: resp.Error}
	}

	// 转换消息格式
	msgs = make([]definition.Msg, len(resp.Msgs))
	for i, pbMsg := range resp.Msgs {
		msgs[i] = definition.Msg{
			Id:        pbMsg.Id,
			Text:      pbMsg.Text,
			CreatedAt: time.Unix(pbMsg.CreatedAt, 0),
		}
	}

	return msgs, nil
}

// ReadByStream 流式读取指定条数消息，并设置超时时间
func (m *MqClient) ReadByStream(mq string, num int, timeout time.Duration) (<-chan definition.Msg, error) {
	req := &pb.ReadRequest{
		Mq:      mq,
		Num:     int32(num),
		Timeout: int64(timeout.Seconds()),
	}

	stream, err := m.client.ReadByStream(m.ctx, req)
	if err != nil {
		return nil, err
	}

	// 创建消息通道
	msgChan := make(chan definition.Msg, num)

	// 在 goroutine 中接收流式消息
	go func() {
		defer close(msgChan)

		for {
			pbMsg, err := stream.Recv()
			if err != nil {
				// 流结束
				return
			}

			msg := definition.Msg{
				Id:        pbMsg.Id,
				Text:      pbMsg.Text,
				CreatedAt: time.Unix(pbMsg.CreatedAt, 0),
			}

			msgChan <- msg
		}
	}()

	return msgChan, nil
}

// 从队列中读取指定条数消息并删除
func (m *MqClient) Pop(mq string, num int) (msgs []definition.Msg, err error) {
	req := &pb.PopRequest{Mq: mq, Num: int32(num)}
	resp, err := m.client.Pop(m.ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, &RpcError{Message: resp.Error}
	}

	// 转换消息格式
	msgs = make([]definition.Msg, len(resp.Msgs))
	for i, pbMsg := range resp.Msgs {
		msgs[i] = definition.Msg{
			Id:        pbMsg.Id,
			Text:      pbMsg.Text,
			CreatedAt: time.Unix(pbMsg.CreatedAt, 0),
		}
	}

	return msgs, nil
}

// 从队列中删除指定消息
func (m *MqClient) Delete(mq string, id uint64) (err error) {
	req := &pb.DeleteRequest{Mq: mq, Id: id}
	resp, err := m.client.Delete(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

// 清空队列
func (m *MqClient) Drop(mq string) (err error) {
	req := &pb.DropRequest{Mq: mq}
	resp, err := m.client.Drop(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

// 将消息存档
func (m *MqClient) Active(mq string, id uint64) (err error) {
	req := &pb.ActiveRequest{Mq: mq, Id: id}
	resp, err := m.client.Active(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

// 获取队列长度
func (m *MqClient) Len(mq string) (int, error) {
	req := &pb.LenRequest{Mq: mq}
	resp, err := m.client.Len(m.ctx, req)
	if err != nil {
		return 0, err
	}
	if resp.Error != "" {
		return 0, &RpcError{Message: resp.Error}
	}
	return int(resp.Length), nil
}

// 设置键值对
func (m *MqClient) SetKeyValue(mq string, key string, value string, expire time.Duration) (err error) {
	req := &pb.SetKeyValueRequest{
		Mq:     mq,
		Key:    key,
		Value:  value,
		Expire: int64(expire.Seconds()),
	}
	resp, err := m.client.SetKeyValue(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

func (m *MqClient) GetKeyValue(mq string, key string) (value *definition.Value, ok bool, err error) {
	req := &pb.GetKeyValueRequest{Mq: mq, Key: key}
	resp, err := m.client.GetKeyValue(m.ctx, req)
	if err != nil {
		return nil, false, err
	}
	if resp.Error != "" {
		return nil, false, &RpcError{Message: resp.Error}
	}

	if !resp.Ok || resp.Value == nil {
		return nil, false, nil
	}

	value = &definition.Value{
		Value:     resp.Value.Value,
		UpdatedAt: time.Unix(resp.Value.UpdatedAt, 0),
		ExpireAt:  time.Unix(resp.Value.ExpireAt, 0),
	}

	return value, true, nil
}

func (m *MqClient) DeleteKeyValue(mq string, key string) (err error) {
	req := &pb.DeleteKeyValueRequest{Mq: mq, Key: key}
	resp, err := m.client.DeleteKeyValue(m.ctx, req)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return &RpcError{Message: resp.Error}
	}
	return nil
}

// Close 关闭连接
func (m *MqClient) Close() error {
	if m.conn != nil {
		return m.conn.Close()
	}
	return nil
}

// ConnectMq 连接gRPC服务器
func ConnectMq(ctx context.Context, addr string, safe bool, auth string) (*MqClient, error) {
	// TLS配置
	tlsConfig := &tls.Config{
		ServerName: extractServerName(addr),
	}
	if !safe {
		tlsConfig.InsecureSkipVerify = true
	}

	// gRPC连接选项
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	// 连接服务器
	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		return nil, err
	}

	client := pb.NewMQClient(conn)

	// 创建带认证信息的上下文
	authCtx := ctx
	if auth != "" {
		expectedAuth := "@" + auth + "@" // 保持与现有认证格式一致
		md := metadata.Pairs("authorization", "Bearer "+expectedAuth)
		authCtx = metadata.NewOutgoingContext(ctx, md)
	}

	mqClient := &MqClient{
		client: client,
		conn:   conn,
		ctx:    authCtx,
	}

	return mqClient, nil
}

// RpcError 自定义错误类型
type RpcError struct {
	Message string
}

func (e *RpcError) Error() string {
	return e.Message
}

// extractServerName 从地址中提取服务器名称
func extractServerName(addr string) string {
	// 简单实现，实际使用中可能需要更复杂的解析
	if len(addr) > 0 && addr[0] == ':' {
		return "localhost"
	}
	return addr
}
