package main

import (
	"flag"
	"log/slog"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/Rehtt/mq/definition"
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
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err)
	}
	slog.Info("server listen", "addr", listener.Addr().String())
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	<-sigChan

	CloseDB()
	slog.Info("server shutdown")
}
