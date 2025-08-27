package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/Rehtt/mq/api/pb"
	"github.com/Rehtt/mq/definition"
	"github.com/Rehtt/mq/internal/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// GrpcServer gRPC服务器
type GrpcServer struct {
	pb.UnimplementedMQServer
	mqService definition.Mq
	server    *grpc.Server
	listener  net.Listener
}

// NewGrpcServer 创建gRPC服务器
func NewGrpcServer(mqService definition.Mq, creds credentials.TransportCredentials, unaryInterceptors []grpc.UnaryServerInterceptor, streamInterceptors []grpc.StreamServerInterceptor) *GrpcServer {
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
	}

	// 添加一元拦截器
	if len(unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}

	// 添加流式拦截器
	if len(streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(streamInterceptors...))
	}

	server := grpc.NewServer(opts...)
	grpcServer := &GrpcServer{
		mqService: mqService,
		server:    server,
	}

	pb.RegisterMQServer(server, grpcServer)
	return grpcServer
}

// Start 启动服务器
func (s *GrpcServer) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = listener
	slog.Info("gRPC server starting", "addr", listener.Addr().String())

	return s.server.Serve(listener)
}

// Stop 停止服务器
func (s *GrpcServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}

// CreateMq 创建队列
func (s *GrpcServer) CreateMq(ctx context.Context, req *pb.CreateMqRequest) (*pb.CreateMqResponse, error) {
	err := s.mqService.CreateMq(req.Mq)
	resp := &pb.CreateMqResponse{}
	if err != nil {
		resp.Error = err.Error()
		// 根据业务错误类型返回相应的gRPC状态码
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeAlreadyExists:
				return resp, status.Error(codes.AlreadyExists, be.Message)
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// DeleteMq 删除队列
func (s *GrpcServer) DeleteMq(ctx context.Context, req *pb.DeleteMqRequest) (*pb.DeleteMqResponse, error) {
	err := s.mqService.DeleteMq(req.Mq)
	resp := &pb.DeleteMqResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// Push 向队列里添加消息
func (s *GrpcServer) Push(ctx context.Context, req *pb.PushRequest) (*pb.PushResponse, error) {
	id, err := s.mqService.Push(req.Mq, req.Msg)
	resp := &pb.PushResponse{Id: id}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// Read 读取指定条数消息，并设置超时时间
func (s *GrpcServer) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	timeout := time.Duration(req.Timeout) * time.Second
	msgs, err := s.mqService.Read(req.Mq, int(req.Num), timeout)

	resp := &pb.ReadResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
		return resp, nil
	}

	// 转换消息格式
	pbMsgs := make([]*pb.Msg, len(msgs))
	for i, msg := range msgs {
		pbMsgs[i] = &pb.Msg{
			Id:        msg.Id,
			Text:      msg.Text,
			CreatedAt: msg.CreatedAt.Unix(),
		}
	}
	resp.Msgs = pbMsgs

	return resp, nil
}

// Pop 从队列中读取指定条数消息并删除
func (s *GrpcServer) Pop(ctx context.Context, req *pb.PopRequest) (*pb.PopResponse, error) {
	msgs, err := s.mqService.Pop(req.Mq, int(req.Num))

	resp := &pb.PopResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
		return resp, nil
	}

	// 转换消息格式
	pbMsgs := make([]*pb.Msg, len(msgs))
	for i, msg := range msgs {
		pbMsgs[i] = &pb.Msg{
			Id:        msg.Id,
			Text:      msg.Text,
			CreatedAt: msg.CreatedAt.Unix(),
		}
	}
	resp.Msgs = pbMsgs

	return resp, nil
}

// Delete 从队列中删除指定消息
func (s *GrpcServer) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	err := s.mqService.Delete(req.Mq, req.Id)
	resp := &pb.DeleteResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeNotFound:
				return resp, status.Error(codes.NotFound, be.Message)
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// Drop 清空队列
func (s *GrpcServer) Drop(ctx context.Context, req *pb.DropRequest) (*pb.DropResponse, error) {
	err := s.mqService.Drop(req.Mq)
	resp := &pb.DropResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// Active 将消息存档
func (s *GrpcServer) Active(ctx context.Context, req *pb.ActiveRequest) (*pb.ActiveResponse, error) {
	err := s.mqService.Active(req.Mq, req.Id)
	resp := &pb.ActiveResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeNotFound:
				return resp, status.Error(codes.NotFound, be.Message)
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// Len 获取队列长度
func (s *GrpcServer) Len(ctx context.Context, req *pb.LenRequest) (*pb.LenResponse, error) {
	length, err := s.mqService.Len(req.Mq)
	resp := &pb.LenResponse{Length: int32(length)}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// SetKeyValue 设置键值对
func (s *GrpcServer) SetKeyValue(ctx context.Context, req *pb.SetKeyValueRequest) (*pb.SetKeyValueResponse, error) {
	expire := time.Duration(req.Expire) * time.Second
	err := s.mqService.SetKeyValue(req.Mq, req.Key, req.Value, expire)
	resp := &pb.SetKeyValueResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// GetKeyValue 获取键值对
func (s *GrpcServer) GetKeyValue(ctx context.Context, req *pb.GetKeyValueRequest) (*pb.GetKeyValueResponse, error) {
	value, ok, err := s.mqService.GetKeyValue(req.Mq, req.Key)

	resp := &pb.GetKeyValueResponse{Ok: ok}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
		return resp, nil
	}

	if ok && value != nil {
		resp.Value = &pb.Value{
			Value:     value.Value,
			UpdatedAt: value.UpdatedAt.Unix(),
			ExpireAt:  value.ExpireAt.Unix(),
		}
	}

	return resp, nil
}

// DeleteKeyValue 删除键值对
func (s *GrpcServer) DeleteKeyValue(ctx context.Context, req *pb.DeleteKeyValueRequest) (*pb.DeleteKeyValueResponse, error) {
	err := s.mqService.DeleteKeyValue(req.Mq, req.Key)
	resp := &pb.DeleteKeyValueResponse{}
	if err != nil {
		resp.Error = err.Error()
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return resp, status.Error(codes.InvalidArgument, be.Message)
			}
		}
	}
	return resp, nil
}

// ReadByStream 流式读取消息
func (s *GrpcServer) ReadByStream(req *pb.ReadRequest, stream pb.MQ_ReadByStreamServer) error {
	timeout := time.Duration(req.Timeout) * time.Second
	msgs, err := s.mqService.Read(req.Mq, int(req.Num), timeout)

	if err != nil {
		if be, ok := err.(*errors.BusinessError); ok {
			switch be.Type {
			case errors.ErrorTypeInvalidInput:
				return status.Error(codes.InvalidArgument, be.Message)
			case errors.ErrorTypeNotFound:
				return status.Error(codes.NotFound, be.Message)
			default:
				return status.Error(codes.Internal, be.Message)
			}
		}
		return status.Error(codes.Internal, err.Error())
	}

	// 逐个发送消息到流
	for _, msg := range msgs {
		pbMsg := &pb.Msg{
			Id:        msg.Id,
			Text:      msg.Text,
			CreatedAt: msg.CreatedAt.Unix(),
		}

		if err := stream.Send(pbMsg); err != nil {
			return status.Error(codes.Internal, "failed to send message: "+err.Error())
		}
	}

	return nil
}
