package main

import (
	"context"
	"flag"
	"log/slog"
	"net/rpc"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Rehtt/mq/definition"
	quic "github.com/quic-go/quic-go"
)

var (
	addr     = flag.String("addr", ":1234", "server address")
	workPath = flag.String("path", "./", "work path")

	tlsCertFile = flag.String("cert", "cert.pem", "tls cert file")
	tlsKeyFile  = flag.String("key", "key.pem", "tls key file")
)

func main() {
	flag.Parse()

	if !filepath.IsAbs(*tlsCertFile) {
		*tlsCertFile = filepath.Join(*workPath, *tlsCertFile)
	}
	if !filepath.IsAbs(*tlsKeyFile) {
		*tlsKeyFile = filepath.Join(*workPath, *tlsKeyFile)
	}

	if err := OpenDB(*workPath); err != nil {
		panic(err)
	}

	tlsConf, err := InitTlsConfig(*tlsCertFile, *tlsKeyFile)
	if err != nil {
		panic(err)
	}
	listener, err := quic.ListenAddr(*addr, tlsConf, nil)
	if err != nil {
		panic(err)
	}

	rpc.RegisterName(definition.MqRpcName, NewMqRpc())
	slog.Info("server listen", "addr", listener.Addr().String())
	go func() {
		for {
			quicConn, err := listener.Accept(context.Background())
			if err != nil {
				continue
			}
			go func(quicConn quic.Connection) {
				stream, err := quicConn.AcceptStream(context.Background())
				if err != nil {
					return
				}
				rpc.ServeConn(stream)
			}(quicConn)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-sigChan

	CloseDB()
	slog.Info("server shutdown")
}
