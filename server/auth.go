package server

import (
	"context"
	"strings"

	"github.com/Rehtt/mq/internal/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authFlag = "@"

// AuthInterceptor 认证拦截器
type AuthInterceptor struct {
	password string
}

// NewAuthInterceptor 创建认证拦截器
func NewAuthInterceptor(password string) *AuthInterceptor {
	return &AuthInterceptor{
		password: password,
	}
}

// UnaryInterceptor 一元RPC认证拦截器
func (a *AuthInterceptor) UnaryInterceptor() grpc.UnaryServerInterceptor {
	if a.password == "" {
		// 如果没有设置密码，跳过认证
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	expectedAuth := authFlag + a.password + authFlag

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 从metadata中获取认证信息
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "metadata not found")
		}

		// 获取authorization header
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization header not found")
		}

		// 验证认证信息
		auth := strings.TrimPrefix(authHeader[0], "Bearer ")
		if auth != expectedAuth {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}

		// 认证成功，继续处理请求
		return handler(ctx, req)
	}
}

// StreamInterceptor 流式RPC认证拦截器
func (a *AuthInterceptor) StreamInterceptor() grpc.StreamServerInterceptor {
	if a.password == "" {
		// 如果没有设置密码，跳过认证
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	expectedAuth := authFlag + a.password + authFlag

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 从metadata中获取认证信息
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "metadata not found")
		}

		// 获取authorization header
		authHeader := md.Get("authorization")
		if len(authHeader) == 0 {
			return status.Error(codes.Unauthenticated, "authorization header not found")
		}

		// 验证认证信息
		auth := strings.TrimPrefix(authHeader[0], "Bearer ")
		if auth != expectedAuth {
			return status.Error(codes.Unauthenticated, "invalid credentials")
		}

		// 认证成功，继续处理请求
		return handler(srv, ss)
	}
}

// ValidateAuth 验证认证信息（用于单元测试）
func (a *AuthInterceptor) ValidateAuth(auth string) error {
	if a.password == "" {
		return nil
	}

	expectedAuth := authFlag + a.password + authFlag
	if auth != expectedAuth {
		return errors.Auth("invalid credentials")
	}

	return nil
}
