package main

import (
	"context"
	"flag"
	"log/slog"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rehtt/mq/definition"
	quic "github.com/quic-go/quic-go"
)

var (
	addr     = flag.String("addr", ":1234", "server address")
	workPath = flag.String("path", "./", "work path")
)

func main() {
	flag.Parse()
	if err := OpenDB(*workPath); err != nil {
		panic(err)
	}

	rpc.RegisterName(definition.MqRpcName, NewMqRpc())

	listener, err := quic.ListenAddr(*addr, GenerateTLSConfig(), nil)
	if err != nil {
		panic(err)
	}
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
