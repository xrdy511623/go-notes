package configparse

/*
性能对比：不同配置格式的解析性能

Go 项目常用的配置格式：JSON、YAML、TOML、环境变量。
本实验对比它们的解析速度和内存分配。

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .

关键结论：
  - JSON 解析最快（标准库原生支持，高度优化）
  - TOML 次之
  - YAML 略慢于 TOML
  - 环境变量最快（直接 os.Getenv，无需解析）
  - 在实际应用中，配置解析只发生在启动时，性能差异可忽略
  - 选择格式应以可读性和可维护性为主，而非性能
*/

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AppConfig 应用配置结构
type AppConfig struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Log      LogConfig      `json:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ReadTimeout  int    `json:"read_timeout"`
	WriteTimeout int    `json:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	MaxConns int    `json:"max_conns"`
	MaxIdle  int    `json:"max_idle"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	PoolSize int    `json:"pool_size"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

// SampleJSON 返回示例 JSON 配置
func SampleJSON() []byte {
	return []byte(`{
  "server": {"host": "0.0.0.0", "port": 8080, "read_timeout": 30, "write_timeout": 30},
  "database": {"host": "localhost", "port": 5432, "name": "myapp", "user": "postgres", "password": "secret", "max_conns": 20, "max_idle": 5},
  "redis": {"host": "localhost", "port": 6379, "password": "", "db": 0, "pool_size": 10},
  "log": {"level": "info", "format": "json", "output": "stdout"}
}`)
}

// SampleYAMLLike 返回模拟 YAML 格式的键值对（用简化解析模拟 YAML 开销）
func SampleYAMLLike() []byte {
	return []byte(`server.host: 0.0.0.0
server.port: 8080
server.read_timeout: 30
server.write_timeout: 30
database.host: localhost
database.port: 5432
database.name: myapp
database.user: postgres
database.password: secret
database.max_conns: 20
database.max_idle: 5
redis.host: localhost
redis.port: 6379
redis.password:
redis.db: 0
redis.pool_size: 10
log.level: info
log.format: json
log.output: stdout`)
}

// ParseJSON 解析 JSON 配置
func ParseJSON(data []byte) (AppConfig, error) {
	var cfg AppConfig
	err := json.Unmarshal(data, &cfg)
	return cfg, err
}

// ParseKeyValue 模拟解析 YAML/TOML（简化为键值对解析）
func ParseKeyValue(data []byte) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}

// ParseEnvVars 模拟从环境变量解析配置
func ParseEnvVars(prefix string, keys []string) map[string]string {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		envKey := fmt.Sprintf("%s_%s", prefix, strings.ToUpper(key))
		result[key] = os.Getenv(envKey)
	}
	return result
}

// ParseEnvVarsDirect 直接解析预定义的环境变量（不经过 os.Getenv 的开销）
func ParseEnvVarsDirect(envs map[string]string) AppConfig {
	port, _ := strconv.Atoi(envs["SERVER_PORT"])
	dbPort, _ := strconv.Atoi(envs["DB_PORT"])
	maxConns, _ := strconv.Atoi(envs["DB_MAX_CONNS"])

	return AppConfig{
		Server:   ServerConfig{Host: envs["SERVER_HOST"], Port: port},
		Database: DatabaseConfig{Host: envs["DB_HOST"], Port: dbPort, MaxConns: maxConns},
	}
}
