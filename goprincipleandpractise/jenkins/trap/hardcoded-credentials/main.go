package hardcodedcredentials

/*
陷阱：Jenkinsfile 中硬编码密钥

问题说明：
  在 Jenkinsfile 中直接写死密码、Token、API Key 等敏感信息，
  会导致密钥随代码提交到 Git 仓库，任何有代码访问权限的人都能看到。

  常见的硬编码形式：

  1. 直接写密码
     sh 'docker login -u admin -p MyP@ssw0rd registry.example.com'

  2. 环境变量中写死 Token
     environment {
         SLACK_TOKEN = 'xoxb-1234567890-abcdef'
         DB_PASSWORD = 'root123'
     }

  3. 在 sh 步骤中拼接密钥
     sh "curl -H 'Authorization: Bearer sk-proj-xxxxx' https://api.example.com"

后果：
  - 密钥泄漏到 Git 历史（即使后来删除，history 中仍存在）
  - Jenkins 构建日志中明文显示密钥
  - 无法按环境区分凭据（staging/production 用同一个）
  - 无法轮换密钥（需要改代码 + 提交 + 审批）

正确做法：
  使用 Jenkins Credentials Store + withCredentials 块：

  // ✅ 正确：引用 Jenkins 凭据
  withCredentials([
      usernamePassword(
          credentialsId: 'docker-registry',
          usernameVariable: 'DOCKER_USER',
          passwordVariable: 'DOCKER_PASS'
      )
  ]) {
      sh 'echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin'
  }

  withCredentials([string(credentialsId: 'slack-token', variable: 'TOKEN')]) {
      sh 'curl -H "Authorization: Bearer $TOKEN" https://api.example.com'
  }

  好处：
  - 密钥不进代码仓库
  - 构建日志中自动掩码为 ****
  - 可按 Folder 隔离不同环境的凭据
  - 轮换密钥只需在 Jenkins UI 修改，无需改代码
*/

import "fmt"

// BadJenkinsfile 展示硬编码密钥的错误 Jenkinsfile
func BadJenkinsfile() string {
	return `// ❌ 错误：密钥硬编码在 Jenkinsfile 中
pipeline {
    agent any
    environment {
        DOCKER_USER = 'admin'
        DOCKER_PASS = 'MyP@ssw0rd'           // 密码明文！
        SLACK_TOKEN = 'xoxb-1234567890'       // Token 明文！
        DB_PASSWORD = 'root123'               // 数据库密码明文！
    }
    stages {
        stage('Deploy') {
            steps {
                sh 'docker login -u $DOCKER_USER -p $DOCKER_PASS registry.example.com'
                sh 'curl -H "Authorization: Bearer $SLACK_TOKEN" ...'
            }
        }
    }
}`
}

// GoodJenkinsfile 展示使用 Jenkins Credentials 的正确做法
func GoodJenkinsfile() string {
	return `// ✅ 正确：使用 Jenkins Credentials Store
pipeline {
    agent any
    stages {
        stage('Deploy') {
            steps {
                withCredentials([
                    usernamePassword(
                        credentialsId: 'docker-registry',
                        usernameVariable: 'DOCKER_USER',
                        passwordVariable: 'DOCKER_PASS'
                    )
                ]) {
                    // 密钥通过环境变量注入，日志自动掩码
                    sh 'echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin'
                }
                withCredentials([string(credentialsId: 'slack-token', variable: 'TOKEN')]) {
                    sh 'curl -H "Authorization: Bearer $TOKEN" ...'
                }
            }
        }
    }
}`
}

// CredentialTypes 列出 Jenkins 支持的凭据类型及用途
func CredentialTypes() map[string]string {
	return map[string]string{
		"Username with password":        "Git 认证、Docker Registry、数据库",
		"Secret text":                   "API Key、Token、Webhook URL",
		"Secret file":                   "kubeconfig、TLS 证书、配置文件",
		"SSH Username with private key": "Git SSH 拉取、服务器 SSH 部署",
		"Certificate":                   "客户端 TLS 证书",
	}
}

// PrintCredentialBestPractices 打印凭据管理最佳实践
func PrintCredentialBestPractices() {
	fmt.Println("=== Jenkins 凭据管理最佳实践 ===")
	fmt.Println()
	fmt.Println("1. 永远不要在 Jenkinsfile 中硬编码任何密钥")
	fmt.Println("2. 使用 withCredentials 块引用 Jenkins Credentials Store")
	fmt.Println("3. 按 Folder 隔离不同团队/环境的凭据")
	fmt.Println("4. 定期轮换凭据，启用审计日志")
	fmt.Println("5. 使用 --password-stdin 避免命令行参数暴露密码")
	fmt.Println()
	fmt.Println("凭据类型：")
	for typ, usage := range CredentialTypes() {
		fmt.Printf("  %-35s → %s\n", typ, usage)
	}
}
