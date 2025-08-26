package server

import (
	"context"
	"testing"

	"github.com/Rehtt/mq/internal/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_ValidateAuth(t *testing.T) {
	tests := []struct {
		name     string
		password string
		auth     string
		wantErr  bool
	}{
		{
			name:     "no password set - should pass any auth",
			password: "",
			auth:     "any-auth",
			wantErr:  false,
		},
		{
			name:     "correct auth",
			password: "secret",
			auth:     "@secret@",
			wantErr:  false,
		},
		{
			name:     "incorrect auth",
			password: "secret",
			auth:     "@wrong@",
			wantErr:  true,
		},
		{
			name:     "malformed auth",
			password: "secret",
			auth:     "malformed",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewAuthInterceptor(tt.password)
			err := interceptor.ValidateAuth(tt.auth)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				if be, ok := err.(*errors.BusinessError); ok {
					if be.Type != errors.ErrorTypeAuth {
						t.Errorf("expected auth error type, got %v", be.Type)
					}
				} else {
					t.Errorf("expected BusinessError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestAuthInterceptor_UnaryInterceptor(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		metadata     map[string]string
		wantErr      bool
		expectedCode codes.Code
	}{
		{
			name:     "no password - should pass without metadata",
			password: "",
			metadata: nil,
			wantErr:  false,
		},
		{
			name:         "no metadata",
			password:     "secret",
			metadata:     nil,
			wantErr:      true,
			expectedCode: codes.Unauthenticated,
		},
		{
			name:         "no authorization header",
			password:     "secret",
			metadata:     map[string]string{"other": "value"},
			wantErr:      true,
			expectedCode: codes.Unauthenticated,
		},
		{
			name:     "correct authorization",
			password: "secret",
			metadata: map[string]string{"authorization": "Bearer @secret@"},
			wantErr:  false,
		},
		{
			name:         "incorrect authorization",
			password:     "secret",
			metadata:     map[string]string{"authorization": "Bearer @wrong@"},
			wantErr:      true,
			expectedCode: codes.Unauthenticated,
		},
		{
			name:     "authorization without Bearer prefix",
			password: "secret",
			metadata: map[string]string{"authorization": "@secret@"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewAuthInterceptor(tt.password)

			// 创建上下文和metadata
			ctx := context.Background()
			if tt.metadata != nil {
				md := metadata.New(tt.metadata)
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// 模拟handler
			handlerCalled := false
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				handlerCalled = true
				return "response", nil
			}

			// 获取拦截器函数
			interceptorFunc := interceptor.UnaryInterceptor()

			// 调用拦截器
			resp, err := interceptorFunc(ctx, "request", &grpc.UnaryServerInfo{}, handler)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				// 检查gRPC状态码
				if st, ok := status.FromError(err); ok {
					if st.Code() != tt.expectedCode {
						t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
					}
				} else {
					t.Errorf("expected gRPC status error, got %T", err)
				}

				// Handler不应该被调用
				if handlerCalled {
					t.Errorf("handler should not be called on auth failure")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Handler应该被调用
				if !handlerCalled {
					t.Errorf("handler should be called on auth success")
				}

				// 检查响应
				if resp != "response" {
					t.Errorf("expected response 'response', got %v", resp)
				}
			}
		})
	}
}

func TestAuthInterceptor_StreamInterceptor(t *testing.T) {
	// 为流式RPC测试基本功能
	interceptor := NewAuthInterceptor("secret")

	// 创建模拟ServerStream
	ctx := context.Background()
	md := metadata.New(map[string]string{"authorization": "Bearer @secret@"})
	ctx = metadata.NewIncomingContext(ctx, md)

	stream := &mockServerStream{ctx: ctx}

	handlerCalled := false
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	streamInterceptor := interceptor.StreamInterceptor()
	err := streamInterceptor(nil, stream, &grpc.StreamServerInfo{}, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Errorf("handler should be called on auth success")
	}
}

func TestAuthInterceptor_StreamInterceptor_AuthFailure(t *testing.T) {
	interceptor := NewAuthInterceptor("secret")

	// 创建没有认证信息的ServerStream
	ctx := context.Background()
	stream := &mockServerStream{ctx: ctx}

	handlerCalled := false
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		handlerCalled = true
		return nil
	}

	streamInterceptor := interceptor.StreamInterceptor()
	err := streamInterceptor(nil, stream, &grpc.StreamServerInfo{}, handler)

	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	// 检查gRPC状态码
	if st, ok := status.FromError(err); ok {
		if st.Code() != codes.Unauthenticated {
			t.Errorf("expected code %v, got %v", codes.Unauthenticated, st.Code())
		}
	} else {
		t.Errorf("expected gRPC status error, got %T", err)
	}

	// Handler不应该被调用
	if handlerCalled {
		t.Errorf("handler should not be called on auth failure")
	}
}

// mockServerStream 模拟ServerStream
type mockServerStream struct {
	ctx context.Context
	grpc.ServerStream
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}
