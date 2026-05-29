package config

import (
	"encoding/json"
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

// AppConfig 表示部署时的 Sender 配置。
// 从 JSON/YAML 反序列化后，调用 BuildSenders 批量创建。
type AppConfig struct {
	Senders []SenderEntry `json:"senders"`
}

// SenderEntry 描述一个 Sender 的名称和配置。
type SenderEntry struct {
	Name   string        `json:"name"`
	Config sender.Config `json:"config"`
}

// BuildSenders 根据配置批量创建 Sender。
// 返回以名称为键的 Sender 映射。任何一个工厂失败则立即返回错误。
func BuildSenders(app AppConfig) (map[string]sender.Sender, error) {
	result := make(map[string]sender.Sender, len(app.Senders))
	for _, entry := range app.Senders {
		if _, exists := result[entry.Name]; exists {
			return nil, fmt.Errorf("config: duplicate sender name %q", entry.Name)
		}
		s, err := sender.New(entry.Name, entry.Config)
		if err != nil {
			return nil, fmt.Errorf("config: failed to create sender %q: %w", entry.Name, err)
		}
		result[entry.Name] = s
	}
	return result, nil
}

// ParseAndBuild 从 JSON 字节解析配置并批量创建 Sender。
func ParseAndBuild(data []byte) (map[string]sender.Sender, error) {
	var app AppConfig
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("config: invalid JSON: %w", err)
	}
	return BuildSenders(app)
}
