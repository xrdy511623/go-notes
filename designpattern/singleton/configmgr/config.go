package configmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Config 保存应用程序的全局配置。
// 通过 Instance() 获取唯一实例，首次调用时初始化。
type Config struct {
	AppName    string `json:"app_name"`
	Debug      bool   `json:"debug"`
	ServerPort int    `json:"server_port"`
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
}

var (
	instance *Config
	once     sync.Once
	initErr  error
)

// Instance 返回全局唯一的 Config 实例。
// 首次调用时通过 sync.Once 执行初始化，后续调用直接返回缓存结果。
// 如果初始化失败，error 会被永久缓存——这是 sync.Once 的语义决定的：
// f 只会执行一次，无论成功还是失败。
func Instance() (*Config, error) {
	once.Do(func() {
		instance, initErr = load()
	})
	return instance, initErr
}

// load 从 CONFIG_PATH 环境变量指定的文件加载配置。
// 如果未设置 CONFIG_PATH，返回默认配置。
func load() (*Config, error) {
	cfg := &Config{
		AppName:    "my-app",
		Debug:      false,
		ServerPort: 8080,
		DBHost:     "localhost",
		DBPort:     3306,
	}

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("configmgr: read %s: %w", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("configmgr: parse %s: %w", path, err)
	}

	return cfg, nil
}

// ResetForTesting 重置单例状态，仅用于测试。
// 生产代码不应调用此方法——单例的生命周期应与进程一致。
func ResetForTesting() {
	once = sync.Once{}
	instance = nil
	initErr = nil
}
