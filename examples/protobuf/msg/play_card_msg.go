package msg

import (
	"context"
	"github.com/cihub/seelog"
	"github.com/fanyang1988/tao"
	"github.com/fanyang1988/tao/examples/protobuf/msg/go"
	"github.com/golang/protobuf/proto"
)

// Message defines the echo message.
type PlayCardReq struct {
	Data demo.PlayCardReq
}

// Serialize serializes Message into bytes.
func (em *PlayCardReq) Serialize() ([]byte, error) {
	return proto.Marshal(&em.Data)
}

// MessageNumber returns message type number.
func (em *PlayCardReq) MessageNumber() int32 {
	return 1
}

// DeserializeMessage deserializes bytes into Message.
func DeserializePlayCardReqMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	msg := new(PlayCardReq)
	proto.Unmarshal(data, &msg.Data)
	return msg, nil
}

// ProcessMessage process the logic of echo message.
func ProcessPlayCardReqMessage(ctx context.Context, conn tao.WriteCloser) {
	rep := tao.MessageFromContext(ctx).(*PlayCardReq)
	seelog.Infof("receving message %v\n", rep.Data)
	seelog.Infof("receving message %v\n", rep.Data.GetCard())
	seelog.Infof("receving message %v\n", rep.Data.GetCard().Card)

	rsp := new(PlayCardRsp)
	rsp.Data.Code = proto.Int32(3)
	conn.Write(rsp)
}

// Message defines the echo message.
type PlayCardRsp struct {
	Data demo.PlayCardRsp
}

// Serialize serializes Message into bytes.
func (em *PlayCardRsp) Serialize() ([]byte, error) {
	return proto.Marshal(&em.Data)
}

// MessageNumber returns message type number.
func (em *PlayCardRsp) MessageNumber() int32 {
	return 2
}

// DeserializeMessage deserializes bytes into Message.
func DeserializePlayCardRspMessage(data []byte) (message tao.Message, err error) {
	if data == nil {
		return nil, tao.ErrNilData
	}
	msg := new(PlayCardRsp)
	proto.Unmarshal(data, &msg.Data)
	return msg, nil
}