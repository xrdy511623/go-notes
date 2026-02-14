package roundtrip

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Round-trip Fuzz 模式演示：自定义二进制编解码
//
// 编码格式: [1 字节 type][2 字节 name 长度][name 数据][4 字节 score]
// 总长度 = 1 + 2 + len(Name) + 4

// Record 表示一条简单记录
type Record struct {
	Type  uint8
	Name  string
	Score int32
}

const headerSize = 1 + 2 + 4 // type + nameLen + score

// Encode 将 Record 编码为字节切片
func Encode(r Record) ([]byte, error) {
	if len(r.Name) > 65535 {
		return nil, fmt.Errorf("name too long: %d > 65535", len(r.Name))
	}
	buf := make([]byte, headerSize+len(r.Name))
	buf[0] = r.Type
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(r.Name)))
	copy(buf[3:3+len(r.Name)], r.Name)
	binary.BigEndian.PutUint32(buf[3+len(r.Name):], uint32(r.Score))
	return buf, nil
}

// Decode 将字节切片解码为 Record
func Decode(data []byte) (Record, error) {
	if len(data) < 3 {
		return Record{}, errors.New("data too short for header")
	}
	typ := data[0]
	nameLen := int(binary.BigEndian.Uint16(data[1:3]))
	if len(data) < 3+nameLen+4 {
		return Record{}, fmt.Errorf("data too short: need %d, got %d", 3+nameLen+4, len(data))
	}
	name := string(data[3 : 3+nameLen])
	score := int32(binary.BigEndian.Uint32(data[3+nameLen:]))
	return Record{Type: typ, Name: name, Score: score}, nil
}
