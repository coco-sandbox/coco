package visor

import (
	"encoding/binary"
	"fmt"
	"io"
)

type MessageType uint8

const (
	MsgTypeStart     MessageType = 1
	MsgTypeStop      MessageType = 2
	MsgTypeExec      MessageType = 3
	MsgTypeSignal    MessageType = 4
	MsgTypeStats     MessageType = 5
	MsgTypeReady     MessageType = 6
	MsgTypeError     MessageType = 7
	MsgTypeOutput    MessageType = 8
	MsgTypeExit      MessageType = 9
	MsgTypeResize    MessageType = 10
	MsgTypeHeartbeat MessageType = 11
)

type Message struct {
	Type    MessageType
	Payload []byte
}

func (m *Message) Encode() ([]byte, error) {
	buf := make([]byte, 1+binary.MaxVarintLen64+len(m.Payload))
	buf[0] = byte(m.Type)

	n := binary.PutUvarint(buf[1:], uint64(len(m.Payload)))
	copy(buf[1+n:], m.Payload)

	return buf[:1+n+len(m.Payload)], nil
}

func DecodeMessage(data []byte) (*Message, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("message too short")
	}

	msg := &Message{
		Type: MessageType(data[0]),
	}

	length, n := binary.Uvarint(data[1:])
	if n <= 0 {
		return nil, fmt.Errorf("invalid length")
	}

	if len(data) < 1+n+int(length) {
		return nil, fmt.Errorf("incomplete message")
	}

	msg.Payload = data[1+n : 1+n+int(length)]
	return msg, nil
}

type Frame struct {
	StreamID uint32
	Type     MessageType
	Data     []byte
}

func (f *Frame) WriteTo(w io.Writer) (int64, error) {
	header := make([]byte, 9)
	binary.LittleEndian.PutUint32(header[0:4], f.StreamID)
	header[4] = byte(f.Type)
	binary.LittleEndian.PutUint32(header[5:9], uint32(len(f.Data)))

	n1, err := w.Write(header)
	if err != nil {
		return int64(n1), err
	}

	n2, err := w.Write(f.Data)
	return int64(n1 + n2), err
}

func ReadFrame(r io.Reader) (*Frame, error) {
	header := make([]byte, 9)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}

	streamID := binary.LittleEndian.Uint32(header[0:4])
	msgType := MessageType(header[4])
	length := binary.LittleEndian.Uint32(header[5:9])

	data := make([]byte, length)
	if length > 0 {
		_, err := io.ReadFull(r, data)
		if err != nil {
			return nil, err
		}
	}

	return &Frame{
		StreamID: streamID,
		Type:    msgType,
		Data:    data,
	}, nil
}
