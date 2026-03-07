package byteparser

import (
	"errors"
	"fmt"
)

// 简单的二进制协议解析器 — 演示 []byte Fuzz 发现 panic
//
// 协议格式: [1 字节 version][1 字节 command][2 字节 payload 长度][payload 数据]
// version 必须为 1 或 2

// Message 表示解析后的协议消息
type Message struct {
	Payload []byte
	Version uint8
	Command uint8
}

// ParseMessage 解析二进制协议消息
func ParseMessage(data []byte) (Message, error) {
	if len(data) < 4 {
		return Message{}, errors.New("data too short: need at least 4 bytes header")
	}

	version := data[0]
	if version != 1 && version != 2 {
		return Message{}, fmt.Errorf("unsupported version: %d", version)
	}

	command := data[1]
	payloadLen := int(data[2])<<8 | int(data[3]) // big-endian uint16

	if payloadLen > 4096 {
		return Message{}, fmt.Errorf("payload too large: %d > 4096", payloadLen)
	}

	if len(data) < 4+payloadLen {
		return Message{}, fmt.Errorf("data truncated: need %d, got %d", 4+payloadLen, len(data))
	}

	// 拷贝 payload，避免持有 data 的引用
	payload := make([]byte, payloadLen)
	copy(payload, data[4:4+payloadLen])

	return Message{
		Version: version,
		Command: command,
		Payload: payload,
	}, nil
}

// ValidateMessage 对解析后的消息做业务校验
func ValidateMessage(msg Message) error {
	switch msg.Command {
	case 0x01: // PING: payload 必须为空
		if len(msg.Payload) != 0 {
			return fmt.Errorf("PING command should have empty payload, got %d bytes", len(msg.Payload))
		}
	case 0x02: // DATA: payload 不能为空
		if len(msg.Payload) == 0 {
			return errors.New("DATA command requires non-empty payload")
		}
	case 0x03: // ECHO: 任意 payload
		// ok
	default:
		return fmt.Errorf("unknown command: 0x%02x", msg.Command)
	}
	return nil
}
