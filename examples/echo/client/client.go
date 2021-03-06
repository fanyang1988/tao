package main

import (
	"fmt"
	"net"
	"time"

	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
	"github.com/fanyang1988/tao/examples/echo"
	"github.com/leesper/holmes"
)

func main() {
	tao.Register(echo.Message{}.MessageNumber(), echo.DeserializeMessage, nil)

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

	echo1 := echo.Message{
		Content: "hello, world",
	}
	echo2 := echo.Message2{}
	echo3 := echo.Message3{}

	conn.Start()

	for i := 0; i < 10; i++ {
		time.Sleep(60 * time.Millisecond)
		err := conn.Write(echo1)
		if err != nil {
			holmes.Errorln(err)
		}
		err = conn.Write(echo2)
		if err != nil {
			holmes.Errorln(err)
		}
		err = conn.Write(echo3)
		if err != nil {
			holmes.Errorln(err)
		}
	}
	holmes.Debugln("hello")
	conn.Close()
}
