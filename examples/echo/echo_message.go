package echo

import (
	"context"

	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
)

// Message defines the echo message.
type Message struct {
	Content string
}

// Serialize serializes Message into bytes.
func (em Message) Serialize() ([]byte, error) {
	return []byte("1111111"), nil
}

// MessageNumber returns message type number.
func (em Message) MessageNumber() int32 {
	return 1
}

// DeserializeMessage deserializes bytes into Message.
func DeserializeMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	msg := string(data)
	echo := Message{
		Content: msg,
	}
	return echo, nil
}

// ProcessMessage process the logic of echo message.
func ProcessMessage(ctx context.Context, conn tao.WriteCloser) {
	msg := tao.MessageFromContext(ctx).(Message)
	seelog.Infof("1111111 %v", msg)
	conn.Write(msg)
}

// Message defines the echo message.
type Message2 struct {
	Content []byte
}

// Serialize serializes Message into bytes.
func (em Message2) Serialize() ([]byte, error) {
	return []byte("22222"), nil
}

// MessageNumber returns message type number.
func (em Message2) MessageNumber() int32 {
	return 2
}

// DeserializeMessage deserializes bytes into Message.
func DeserializeMessage2(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	echo := Message2{}
	return echo, nil
}

// ProcessMessage process the logic of echo message.
func ProcessMessage2(ctx context.Context, conn tao.WriteCloser) {
	msg := tao.MessageFromContext(ctx).(Message2)
	seelog.Infof("22222 %v", msg)
	conn.Write(msg)
}

// Message defines the echo message.
type Message3 struct {
	Content []byte
}

// Serialize serializes Message into bytes.
func (em Message3) Serialize() ([]byte, error) {
	return []byte(em.Content), nil
}

// MessageNumber returns message type number.
func (em Message3) MessageNumber() int32 {
	return 3
}

// DeserializeMessage deserializes bytes into Message.
func DeserializeMessage3(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	echo := Message3{}
	return echo, nil
}

// ProcessMessage process the logic of echo message.
func ProcessMessage3(ctx context.Context, conn tao.WriteCloser) {
	msg := tao.MessageFromContext(ctx).(Message3)
	seelog.Infof("333333 %v", msg)
	conn.Write(msg)
}
