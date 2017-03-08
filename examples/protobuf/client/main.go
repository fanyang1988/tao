package main

import (
	"fmt"
	"net"
	"time"
	"context"

	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
	"github.com/fanyang1988/tao/examples/echo"
	"github.com/leesper/holmes"
	"github.com/fanyang1988/tao/examples/protobuf/msg"
	"github.com/fanyang1988/tao/examples/protobuf/msg/go"
)

func main() {
	p := msg.PlayCardRsp{}
	tao.Register(p.MessageNumber(), msg.DeserializePlayCardRspMessage, ProcessMessage)
	p2 := msg.PlayCardReq{}
	tao.Register(p2.MessageNumber(), msg.DeserializePlayCardReqMessage, nil)

	c, err := net.Dial("tcp", "127.0.0.1:12345")
	if err != nil {
		seelog.Criticalf(err.Error())
	}

	onConnect := tao.OnConnectOption(func(conn tao.WriteCloser) bool {
		seelog.Infof("on connect")
		return true
	})

	onError := tao.OnErrorOption(func(conn tao.WriteCloser) {
		seelog.Infof("on error")
	})

	onClose := tao.OnCloseOption(func(conn tao.WriteCloser) {
		seelog.Infof("on close")
	})

	onMessage := tao.OnMessageOption(func(msg tao.Message, conn tao.WriteCloser) {
		echo := msg.(echo.Message)
		fmt.Printf("%s\n", echo.Content)
	})

	conn := tao.NewClientConn(0, c, onConnect, onError, onClose, onMessage)

	req := &msg.PlayCardReq{
	}
	req.Data.Card = &demo.Cards{
		Card:[]int32{1,2,3},
	}

	conn.Start()

	for i := 0; i < 10; i++ {
		time.Sleep(60 * time.Millisecond)
		err := conn.Write(req)
		if err != nil {
			holmes.Errorln(err)
		}
	}



	holmes.Debugln("hello")
	conn.Close()
}

// ProcessPingPongMessage handles business logic.
func ProcessMessage(ctx context.Context, conn tao.WriteCloser) {
	re := tao.MessageFromContext(ctx).(*msg.PlayCardRsp)
	seelog.Infof("resp %d",re.Data.GetCode())
}
