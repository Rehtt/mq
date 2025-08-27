package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Rehtt/mq/internal/mq"
	"github.com/Rehtt/mq/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	addr     = flag.String("addr", ":1234", "server address")
	workPath = flag.String("path", "./", "work path")

	tlsCertFile = flag.String("cert", "cert.pem", "tls cert file")
	tlsKeyFile  = flag.String("key", "key.pem", "tls key file")

	password = flag.String("password", "", "password")
)

func main() {
	flag.Parse()

	showInfo()

	// 确保TLS证书路径是绝对路径
	if !filepath.IsAbs(*tlsCertFile) {
		*tlsCertFile = filepath.Join(*workPath, *tlsCertFile)
	}
	if !filepath.IsAbs(*tlsKeyFile) {
		*tlsKeyFile = filepath.Join(*workPath, *tlsKeyFile)
	}

	// 初始化数据库仓库
	repo, err := mq.NewSQLiteRepository(*workPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	// 创建MQ服务
	mqService := mq.NewService(repo)

	// 初始化TLS配置
	tlsConf, err := InitTlsConfig(*tlsCertFile, *tlsKeyFile)
	if err != nil {
		slog.Error("failed to initialize TLS config", "error", err)
		os.Exit(1)
	}

	creds := credentials.NewTLS(tlsConf)

	// 创建认证拦截器
	authInterceptor := server.NewAuthInterceptor(*password)

	// 创建并启动gRPC服务器
	grpcServer := server.NewGrpcServer(
		mqService,
		creds,
		[]grpc.UnaryServerInterceptor{authInterceptor.UnaryInterceptor()},
		[]grpc.StreamServerInterceptor{authInterceptor.StreamInterceptor()},
	)

	// 在goroutine中启动服务器
	errCh := make(chan error, 1)
	go func() {
		if err := grpcServer.Start(*addr); err != nil {
			errCh <- err
		}
	}()

	// 等待信号或启动错误
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	case sig := <-sigCh:
		slog.Info("received signal, shutting down", "signal", sig)
	}

	// 优雅关闭
	grpcServer.Stop()
	slog.Info("server shutdown completed")
}
