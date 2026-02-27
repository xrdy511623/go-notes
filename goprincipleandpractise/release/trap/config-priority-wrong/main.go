package configprioritywrong

/*
陷阱：配置优先级反转

问题说明：
  12-Factor App 原则要求配置优先级为：
    flag > 环境变量 > 配置文件 > 默认值

  如果优先级搞反（配置文件覆盖环境变量），会导致：
  1. Docker 部署时环境变量不生效
  2. Kubernetes ConfigMap/Secret 无法覆盖配置
  3. CI/CD 中无法通过环境变量切换配置

  错误场景：
    # 配置文件 config.yaml
    database:
      host: db.production.local  # 生产数据库

    # Kubernetes 部署时设置环境变量
    env:
      - name: APP_DATABASE_HOST
        value: db.staging.local   # 想连 staging 数据库

    # 但配置文件优先级更高，环境变量被忽略
    # 结果：staging 环境连上了生产数据库！

正确做法：
  viper 默认优先级就是正确的（flag > env > config > default），
  但要确保调用顺序正确：

  viper.SetDefault(...)            // 1. 默认值
  viper.SetConfigFile(...)         // 2. 配置文件
  viper.ReadInConfig()
  viper.SetEnvPrefix("APP")       // 3. 环境变量
  viper.AutomaticEnv()
  viper.BindPFlags(pflag.CommandLine) // 4. 命令行 flag
*/

import "fmt"

// Config 应用配置
type Config struct {
	DatabaseHost string
	DatabasePort int
	ServerPort   int
}

// WrongPriority 模拟错误的配置优先级
// 配置文件覆盖环境变量
func WrongPriority(envHost, fileHost string) Config {
	// 先读环境变量
	host := envHost
	// 再读配置文件 → 覆盖了环境变量！
	if fileHost != "" {
		host = fileHost // 错误：配置文件不应该覆盖环境变量
	}
	return Config{DatabaseHost: host, DatabasePort: 5432, ServerPort: 8080}
}

// CorrectPriority 模拟正确的配置优先级
// 环境变量覆盖配置文件
func CorrectPriority(defaultHost, fileHost, envHost, flagHost string) Config {
	// 按优先级从低到高覆盖
	host := defaultHost
	if fileHost != "" {
		host = fileHost
	}
	if envHost != "" {
		host = envHost // 环境变量覆盖配置文件
	}
	if flagHost != "" {
		host = flagHost // flag 覆盖一切
	}
	return Config{DatabaseHost: host, DatabasePort: 5432, ServerPort: 8080}
}

// DemonstrateProblem 演示优先级错误导致的问题
func DemonstrateProblem() {
	fmt.Println("=== 配置优先级问题演示 ===")
	fmt.Println()

	// 场景：staging 环境想连 staging 数据库
	envHost := "db.staging.local"
	fileHost := "db.production.local"

	wrong := WrongPriority(envHost, fileHost)
	fmt.Printf("❌ 错误优先级：连接到 %s（生产数据库！）\n", wrong.DatabaseHost)

	correct := CorrectPriority("localhost", fileHost, envHost, "")
	fmt.Printf("✅ 正确优先级：连接到 %s（staging 数据库）\n", correct.DatabaseHost)

	fmt.Println()
	fmt.Println("正确的优先级顺序：")
	fmt.Println("  flag > 环境变量 > 配置文件 > 默认值")
}
