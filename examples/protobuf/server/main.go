package main

import (
	"net"
	"runtime"

	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
	"github.com/fanyang1988/tao/examples/protobuf/msg"
	"github.com/fanyang1988/tao/logger"
)

// EchoServer represents the echo server.
type EchoServer struct {
	*tao.Server
}

// NewEchoServer returns an EchoServer.
func NewEchoServer() *EchoServer {
	onConnect := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		seelog.Infof("on connect")
		return true
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		seelog.Infof("closing client")
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		seelog.Infof("on error")
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, conn tao.WriteCloser) {
		seelog.Infof("receving message")
	})

	return &EchoServer{
		tao.NewServer(logger.NewSeeLogLogger(),
			onConnect, onClose, onError, onMessage),
	}
}

func main() {
	defer seelog.Flush()

	runtime.GOMAXPROCS(runtime.NumCPU())

	n := msg.PlayCardReq{}
	tao.Register(n.MessageNumber(),
		msg.DeserializePlayCardReqMessage,
		msg.ProcessPlayCardReqMessage)

	l, err := net.Listen("tcp", ":12345")
	if err != nil {
		seelog.Criticalf("listen error %v", err)
	}
	echoServer := NewEchoServer()
	defer echoServer.Stop()

	echoServer.Start(l)
}
