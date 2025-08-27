package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/Rehtt/mq/api/pb"
	"github.com/Rehtt/mq/definition"
	"github.com/Rehtt/mq/internal/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// mockMqService 模拟MQ服务
type mockMqService struct {
	createMqFunc       func(string) error
	deleteMqFunc       func(string) error
	pushFunc           func(string, string) (uint64, error)
	readFunc           func(string, int, time.Duration) ([]definition.Msg, error)
	popFunc            func(string, int) ([]definition.Msg, error)
	deleteFunc         func(string, uint64) error
	dropFunc           func(string) error
	activeFunc         func(string, uint64) error
	lenFunc            func(string) (int, error)
	setKeyValueFunc    func(string, string, string, time.Duration) error
	getKeyValueFunc    func(string, string) (*definition.Value, bool, error)
	deleteKeyValueFunc func(string, string) error
}

func (m *mockMqService) CreateMq(mq string) error {
	if m.createMqFunc != nil {
		return m.createMqFunc(mq)
	}
	return nil
}

func (m *mockMqService) DeleteMq(mq string) error {
	if m.deleteMqFunc != nil {
		return m.deleteMqFunc(mq)
	}
	return nil
}

func (m *mockMqService) Push(mq string, msg string) (uint64, error) {
	if m.pushFunc != nil {
		return m.pushFunc(mq, msg)
	}
	return 1, nil
}

func (m *mockMqService) Read(mq string, num int, timeout time.Duration) ([]definition.Msg, error) {
	if m.readFunc != nil {
		return m.readFunc(mq, num, timeout)
	}
	return []definition.Msg{
		{Id: 1, Text: "test message", CreatedAt: time.Now()},
	}, nil
}

func (m *mockMqService) Pop(mq string, num int) ([]definition.Msg, error) {
	if m.popFunc != nil {
		return m.popFunc(mq, num)
	}
	return []definition.Msg{
		{Id: 1, Text: "test message", CreatedAt: time.Now()},
	}, nil
}

func (m *mockMqService) Delete(mq string, id uint64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(mq, id)
	}
	return nil
}

func (m *mockMqService) Drop(mq string) error {
	if m.dropFunc != nil {
		return m.dropFunc(mq)
	}
	return nil
}

func (m *mockMqService) Active(mq string, id uint64) error {
	if m.activeFunc != nil {
		return m.activeFunc(mq, id)
	}
	return nil
}

func (m *mockMqService) Len(mq string) (int, error) {
	if m.lenFunc != nil {
		return m.lenFunc(mq)
	}
	return 1, nil
}

func (m *mockMqService) SetKeyValue(mq string, key string, value string, expire time.Duration) error {
	if m.setKeyValueFunc != nil {
		return m.setKeyValueFunc(mq, key, value, expire)
	}
	return nil
}

func (m *mockMqService) GetKeyValue(mq string, key string) (*definition.Value, bool, error) {
	if m.getKeyValueFunc != nil {
		return m.getKeyValueFunc(mq, key)
	}
	return &definition.Value{
		Value:     "test-value",
		UpdatedAt: time.Now(),
		ExpireAt:  time.Time{},
	}, true, nil
}

func (m *mockMqService) DeleteKeyValue(mq string, key string) error {
	if m.deleteKeyValueFunc != nil {
		return m.deleteKeyValueFunc(mq, key)
	}
	return nil
}

// setupTestServer 设置测试服务器
func setupTestServer(t *testing.T, mqService definition.Mq, password string) (*grpc.ClientConn, func()) {
	lis := bufconn.Listen(bufSize)

	// 使用insecure凭据进行测试
	creds := insecure.NewCredentials()

	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	if password != "" {
		auth := NewAuthInterceptor(password)
		unaryInterceptors = append(unaryInterceptors, auth.UnaryInterceptor())
		streamInterceptors = append(streamInterceptors, auth.StreamInterceptor())
	}

	// 创建服务器选项
	opts := []grpc.ServerOption{}
	if len(unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}
	if len(streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(streamInterceptors...))
	}

	server := grpc.NewServer(opts...)
	grpcServer := &GrpcServer{
		mqService: mqService,
		server:    server,
	}

	pb.RegisterMQServer(server, grpcServer)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()

	// 客户端连接
	conn, err := grpc.Dial("bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(creds))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	cleanup := func() {
		conn.Close()
		server.GracefulStop()
		lis.Close()
	}

	return conn, cleanup
}

func TestGrpcServer_CreateMq(t *testing.T) {
	tests := []struct {
		name        string
		serviceMock func() definition.Mq
		request     *pb.CreateMqRequest
		wantErr     bool
		wantCode    codes.Code
	}{
		{
			name: "successful creation",
			serviceMock: func() definition.Mq {
				return &mockMqService{}
			},
			request: &pb.CreateMqRequest{Mq: "test-queue"},
			wantErr: false,
		},
		{
			name: "invalid input",
			serviceMock: func() definition.Mq {
				return &mockMqService{
					createMqFunc: func(mq string) error {
						return errors.InvalidInput("queue name cannot be empty")
					},
				}
			},
			request:  &pb.CreateMqRequest{Mq: ""},
			wantErr:  true,
			wantCode: codes.InvalidArgument,
		},
		{
			name: "already exists",
			serviceMock: func() definition.Mq {
				return &mockMqService{
					createMqFunc: func(mq string) error {
						return errors.AlreadyExists("queue already exists")
					},
				}
			},
			request:  &pb.CreateMqRequest{Mq: "existing-queue"},
			wantErr:  true,
			wantCode: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, cleanup := setupTestServer(t, tt.serviceMock(), "")
			defer cleanup()

			client := pb.NewMQClient(conn)
			ctx := context.Background()

			resp, err := client.CreateMq(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				if st, ok := status.FromError(err); ok {
					if st.Code() != tt.wantCode {
						t.Errorf("expected code %v, got %v", tt.wantCode, st.Code())
					}
				} else {
					t.Errorf("expected gRPC status error, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if resp == nil {
					t.Errorf("expected response, got nil")
				}
			}
		})
	}
}

func TestGrpcServer_Push(t *testing.T) {
	mockService := &mockMqService{
		pushFunc: func(mq, msg string) (uint64, error) {
			if mq == "test-queue" && msg == "test message" {
				return 123, nil
			}
			return 0, errors.InvalidInput("invalid input")
		},
	}

	conn, cleanup := setupTestServer(t, mockService, "")
	defer cleanup()

	client := pb.NewMQClient(conn)
	ctx := context.Background()

	resp, err := client.Push(ctx, &pb.PushRequest{
		Mq:  "test-queue",
		Msg: "test message",
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if resp.Id != 123 {
		t.Errorf("expected id 123, got %d", resp.Id)
	}
}

func TestGrpcServer_Read(t *testing.T) {
	testTime := time.Now()
	mockService := &mockMqService{
		readFunc: func(mq string, num int, timeout time.Duration) ([]definition.Msg, error) {
			return []definition.Msg{
				{Id: 1, Text: "message1", CreatedAt: testTime},
				{Id: 2, Text: "message2", CreatedAt: testTime},
			}, nil
		},
	}

	conn, cleanup := setupTestServer(t, mockService, "")
	defer cleanup()

	client := pb.NewMQClient(conn)
	ctx := context.Background()

	resp, err := client.Read(ctx, &pb.ReadRequest{
		Mq:      "test-queue",
		Num:     2,
		Timeout: 10,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(resp.Msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(resp.Msgs))
	}

	if resp.Msgs[0].Id != 1 {
		t.Errorf("expected first message id 1, got %d", resp.Msgs[0].Id)
	}

	if resp.Msgs[0].Text != "message1" {
		t.Errorf("expected first message text 'message1', got %s", resp.Msgs[0].Text)
	}
}

func TestGrpcServer_KeyValue(t *testing.T) {
	testTime := time.Now()
	mockService := &mockMqService{
		setKeyValueFunc: func(mq, key, value string, expire time.Duration) error {
			return nil
		},
		getKeyValueFunc: func(mq, key string) (*definition.Value, bool, error) {
			if mq == "test-queue" && key == "test-key" {
				return &definition.Value{
					Value:     "test-value",
					UpdatedAt: testTime,
					ExpireAt:  time.Time{},
				}, true, nil
			}
			return nil, false, nil
		},
		deleteKeyValueFunc: func(mq, key string) error {
			return nil
		},
	}

	conn, cleanup := setupTestServer(t, mockService, "")
	defer cleanup()

	client := pb.NewMQClient(conn)
	ctx := context.Background()

	// Test SetKeyValue
	setResp, err := client.SetKeyValue(ctx, &pb.SetKeyValueRequest{
		Mq:     "test-queue",
		Key:    "test-key",
		Value:  "test-value",
		Expire: 3600,
	})

	if err != nil {
		t.Errorf("unexpected error setting key-value: %v", err)
	}

	if setResp.Error != "" {
		t.Errorf("expected no error, got %s", setResp.Error)
	}

	// Test GetKeyValue
	getResp, err := client.GetKeyValue(ctx, &pb.GetKeyValueRequest{
		Mq:  "test-queue",
		Key: "test-key",
	})

	if err != nil {
		t.Errorf("unexpected error getting key-value: %v", err)
	}

	if !getResp.Ok {
		t.Errorf("expected key to exist")
	}

	if getResp.Value.Value != "test-value" {
		t.Errorf("expected value 'test-value', got %s", getResp.Value.Value)
	}

	// Test DeleteKeyValue
	delResp, err := client.DeleteKeyValue(ctx, &pb.DeleteKeyValueRequest{
		Mq:  "test-queue",
		Key: "test-key",
	})

	if err != nil {
		t.Errorf("unexpected error deleting key-value: %v", err)
	}

	if delResp.Error != "" {
		t.Errorf("expected no error, got %s", delResp.Error)
	}
}

func TestGrpcServer_ReadByStream(t *testing.T) {
	testTime := time.Now()
	mockService := &mockMqService{
		readFunc: func(mq string, num int, timeout time.Duration) ([]definition.Msg, error) {
			return []definition.Msg{
				{Id: 1, Text: "stream-message1", CreatedAt: testTime},
				{Id: 2, Text: "stream-message2", CreatedAt: testTime},
			}, nil
		},
	}

	conn, cleanup := setupTestServer(t, mockService, "")
	defer cleanup()

	client := pb.NewMQClient(conn)
	ctx := context.Background()

	stream, err := client.ReadByStream(ctx, &pb.ReadRequest{
		Mq:      "test-queue",
		Num:     2,
		Timeout: 10,
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// 读取流中的消息
	var messages []*pb.Msg
	for {
		msg, err := stream.Recv()
		if err != nil {
			// 流结束
			break
		}
		messages = append(messages, msg)
	}

	if len(messages) != 2 {
		t.Errorf("expected 2 messages from stream, got %d", len(messages))
	}

	if messages[0].Id != 1 {
		t.Errorf("expected first message id 1, got %d", messages[0].Id)
	}

	if messages[0].Text != "stream-message1" {
		t.Errorf("expected first message text 'stream-message1', got %s", messages[0].Text)
	}

	if messages[1].Id != 2 {
		t.Errorf("expected second message id 2, got %d", messages[1].Id)
	}

	if messages[1].Text != "stream-message2" {
		t.Errorf("expected second message text 'stream-message2', got %s", messages[1].Text)
	}
}

func TestGrpcServer_ReadByStream_WithAuth(t *testing.T) {
	mockService := &mockMqService{
		readFunc: func(mq string, num int, timeout time.Duration) ([]definition.Msg, error) {
			return []definition.Msg{
				{Id: 1, Text: "auth-message", CreatedAt: time.Now()},
			}, nil
		},
	}

	conn, cleanup := setupTestServer(t, mockService, "secret")
	defer cleanup()

	client := pb.NewMQClient(conn)

	// 测试没有认证的情况
	ctx := context.Background()
	stream, err := client.ReadByStream(ctx, &pb.ReadRequest{
		Mq:      "test-queue",
		Num:     1,
		Timeout: 10,
	})

	if err == nil {
		// 尝试从流中读取消息，这时应该会出现认证错误
		_, err = stream.Recv()
		if err == nil {
			t.Errorf("expected authentication error, got nil")
			return
		}
		// 检查是否是认证错误
		if st, ok := status.FromError(err); ok {
			if st.Code() != codes.Unauthenticated {
				t.Errorf("expected Unauthenticated error, got %v", st.Code())
			}
		} else {
			t.Errorf("expected gRPC status error, got %T", err)
		}
	} else {
		// 检查是否是认证错误
		if st, ok := status.FromError(err); ok {
			if st.Code() != codes.Unauthenticated {
				t.Errorf("expected Unauthenticated error, got %v", st.Code())
			}
		} else {
			t.Errorf("expected gRPC status error, got %T", err)
		}
	}

	// 测试有正确认证的情况
	md := metadata.Pairs("authorization", "Bearer @secret@")
	authCtx := metadata.NewOutgoingContext(context.Background(), md)

	stream, err = client.ReadByStream(authCtx, &pb.ReadRequest{
		Mq:      "test-queue",
		Num:     1,
		Timeout: 10,
	})

	if err != nil {
		t.Errorf("unexpected error with valid auth: %v", err)
	}

	// 读取一条消息验证认证成功
	msg, err := stream.Recv()
	if err != nil {
		t.Errorf("failed to receive message: %v", err)
	}

	if msg.Text != "auth-message" {
		t.Errorf("expected message text 'auth-message', got %s", msg.Text)
	}
}
