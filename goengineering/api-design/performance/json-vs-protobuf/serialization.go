// Package serialization 对比 JSON 与 Gob（二进制）序列化的性能。
//
// 注意: 此示例使用 encoding/gob 作为二进制序列化的代理。
// 真实的 Protocol Buffers（google.golang.org/protobuf）通常比 Gob 更快，
// 因为 protobuf 使用代码生成而非反射。本示例旨在展示文本 vs 二进制的差异趋势。
package serialization

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"time"
)

// User 是序列化测试用的数据模型。
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags"`
}

// SampleUser 返回一个测试用例。
func SampleUser() User {
	return User{
		ID:        "usr_000001",
		Name:      "Alice",
		Email:     "alice@example.com",
		Age:       30,
		Active:    true,
		CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Tags:      []string{"admin", "developer", "reviewer"},
	}
}

// SampleUsers 返回 n 个测试用户（模拟列表接口）。
func SampleUsers(n int) []User {
	users := make([]User, n)
	base := SampleUser()
	for i := range n {
		u := base
		u.ID = "usr_" + padInt(i+1)
		users[i] = u
	}
	return users
}

func padInt(n int) string {
	s := ""
	for v := n; v > 0; v /= 10 {
		s = string(rune('0'+v%10)) + s
	}
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}

// MarshalJSON 序列化为 JSON。
func MarshalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

// UnmarshalJSON 从 JSON 反序列化。
func UnmarshalJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// MarshalGob 序列化为 Gob 二进制格式。
func MarshalGob(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalGob 从 Gob 二进制格式反序列化。
func UnmarshalGob(data []byte, v any) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}
