package main

import (
	"context"
	"net"
	"runtime"

	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
	"github.com/fanyang1988/tao/examples/pingpong"
	"github.com/fanyang1988/tao/logger"
)

// PingPongServer defines pingpong server.
type PingPongServer struct {
	*tao.Server
}

// NewPingPongServer returns PingPongServer.
func NewPingPongServer() *PingPongServer {
	onConnect := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		seelog.Infof("on connect")
		return true
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		seelog.Infof("on error")
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		seelog.Infof("closing pingpong client")
	})

	return &PingPongServer{
		tao.NewServer(logger.NewSeeLogLogger(),
			onConnect, onError, onClose),
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer seelog.Flush()
	tao.MonitorOn(12345)
	tao.Register(pingpong.PingPontMessage, pingpong.DeserializeMessage, ProcessPingPongMessage)

	l, err := net.Listen("tcp", ":12346")
	if err != nil {
		seelog.Criticalf("listen error", err)
	}

	server := NewPingPongServer()

	server.Start(l)
}

// ProcessPingPongMessage handles business logic.
func ProcessPingPongMessage(ctx context.Context, conn tao.WriteCloser) {
	ping := tao.MessageFromContext(ctx).(pingpong.Message)
	seelog.Infof("%v", ping.Info)
	rsp := pingpong.Message{
		Info: "pong",
	}
	conn.Write(rsp)
}
