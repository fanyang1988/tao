package main

import (
	"fmt"
	"net"

	"github.com/cihub/seelog"
	"github.com/leesper/holmes"
	"github.com/leesper/tao"
	"github.com/leesper/tao/examples/chat"
)

// ChatServer is the chatting server.
type ChatServer struct {
	*tao.Server
}

// NewChatServer returns a ChatServer.
func NewChatServer() *ChatServer {
	onConnectOption := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		seelog.Infof("on connect")
		return true
	})
	onErrorOption := tao.OnErrorOption(func(conn tao.WriteCloser) {
		seelog.Infof("on error")
	})
	onCloseOption := tao.OnCloseOption(func(conn tao.WriteCloser) {
		seelog.Infof("close chat client")
	})
	return &ChatServer{
		tao.NewServer(onConnectOption, onErrorOption, onCloseOption),
	}
}

func main() {
	defer holmes.Start().Stop()

	tao.Register(chat.ChatMessage, chat.DeserializeMessage, chat.ProcessMessage)

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", 12345))
	if err != nil {
		seelog.Criticalf("listen error", err)
	}
	chatServer := NewChatServer()
	err = chatServer.Start(l)
	if err != nil {
		seelog.Criticalf("start error", err)
	}
}
