package main

import (
	"net"
	"runtime"

	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
	"github.com/fanyang1988/tao/examples/echo"
	"github.com/fanyang1988/tao/logger"
	"github.com/leesper/holmes"
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
	defer holmes.Start().Stop()

	runtime.GOMAXPROCS(1)

	tao.Register(echo.Message{}.MessageNumber(),
		echo.DeserializeMessage,
		echo.ProcessMessage)
	tao.Register(echo.Message2{}.MessageNumber(),
		echo.DeserializeMessage2,
		echo.ProcessMessage2)
	tao.Register(echo.Message3{}.MessageNumber(),
		echo.DeserializeMessage3,
		echo.ProcessMessage3)

	l, err := net.Listen("tcp", ":12345")
	if err != nil {
		holmes.Fatalf("listen error %v", err)
	}
	echoServer := NewEchoServer()
	defer echoServer.Stop()

	echoServer.Start(l)
}
