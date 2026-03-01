# Go 项目 Jenkins Pipeline 详解

> Jenkins 是最流行的自托管 CI/CD 服务器，在企业内网环境中广泛用于 Go 项目的持续集成与持续部署。本文覆盖 Pipeline 语法、Go 项目最佳实践、部署策略，配有反例（trap/）和性能基准（performance/）。

## 目录

1. [Jenkins vs GitHub Actions / GitLab CI](#1-jenkins-vs-github-actions--gitlab-ci)
2. [Pipeline 基础](#2-pipeline-基础)
3. [Go 项目 Jenkinsfile 实战](#3-go-项目-jenkinsfile-实战)
4. [凭据与密钥管理](#4-凭据与密钥管理)
5. [Agent 与构建环境](#5-agent-与构建环境)
6. [Shared Library（共享库）](#6-shared-library共享库)
7. [CD：部署策略](#7-cd部署策略)
8. [Webhook 与触发机制](#8-webhook-与触发机制)
9. [进阶技巧](#9-进阶技巧)

---

## 1 Jenkins vs GitHub Actions / GitLab CI

| 维度 | Jenkins | GitHub Actions | GitLab CI |
|------|---------|---------------|-----------|
| 部署方式 | **自托管**（需运维） | SaaS（GitHub 托管） | SaaS + 可自托管 Runner |
| 配置方式 | `Jenkinsfile`（Groovy DSL） | YAML | YAML |
| 插件生态 | **1800+** 插件（最丰富） | Marketplace Action | 模板库 |
| 学习曲线 | 高（Groovy + 插件体系） | 低 | 中 |
| 并行能力 | `parallel` 块 + 多 Agent | Job 默认并行 | 同 Stage 内 Job 并行 |
| 适用场景 | 企业内网、复杂流水线 | 开源项目、GitHub 生态 | GitLab 全家桶 |
| 成本 | 开源免费（需服务器） | 免费额度 + 按量 | 免费额度 + 按量 |

**什么时候选 Jenkins？**
- 代码在内网 Git（不能访问外网）
- 需要连接内网数据库、私有镜像仓库
- 流水线逻辑复杂，需要 Groovy 的编程能力
- 已有 Jenkins 基础设施

---

## 2 Pipeline 基础

### 2.1 Declarative vs Scripted Pipeline

Jenkins 有两种 Pipeline 语法：

**Declarative Pipeline（推荐）**：
```groovy
// Jenkinsfile (Declarative)
pipeline {
    agent any

    stages {
        stage('Build') {
            steps {
                sh 'go build ./...'
            }
        }
        stage('Test') {
            steps {
                sh 'go test -race ./...'
            }
        }
    }

    post {
        always {
            cleanWs()   // 清理工作空间
        }
    }
}
```

**Scripted Pipeline**：
```groovy
// Jenkinsfile (Scripted)
node {
    stage('Build') {
        checkout scm
        sh 'go build ./...'
    }
    stage('Test') {
        sh 'go test -race ./...'
    }
}
```

| 特性 | Declarative | Scripted |
|------|-------------|----------|
| 语法 | 固定结构（`pipeline { }` 块） | 自由 Groovy 代码 |
| 可读性 | 高（类 YAML 结构化） | 低（需懂 Groovy） |
| 灵活性 | 中（`script { }` 块可嵌入 Groovy） | 高（完全编程） |
| 错误检查 | 语法校验更严格 | 运行时才发现 |
| **推荐** | **日常使用** | 极复杂场景 |

**建议**：始终使用 Declarative Pipeline。需要复杂逻辑时，在 `script { }` 块中嵌入 Groovy，而非整个文件用 Scripted。

### 2.2 核心概念

```
Jenkins Controller
├── Pipeline（流水线）
│   ├── Stage（阶段）— Lint, Test, Build, Deploy
│   │   ├── Step（步骤）— sh, echo, checkout
│   │   └── Step
│   └── Stage
├── Agent（执行节点）— any, label, docker
└── Post（后置动作）— always, success, failure
```

| 概念 | 作用 | 类比 GitHub Actions |
|------|------|-------------------|
| Pipeline | 整个流水线定义 | Workflow |
| Stage | 逻辑阶段 | Job |
| Step | 具体执行步骤 | Step |
| Agent | 执行环境 | runs-on |
| Post | 后置处理 | `if: always()` |

### 2.3 常用 Step

```groovy
steps {
    // 执行 shell 命令
    sh 'go test ./...'

    // 多行 shell
    sh '''
        go test -race -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out
    '''

    // 检出代码
    checkout scm

    // 打印信息
    echo "Building version: ${env.BUILD_NUMBER}"

    // 设置环境变量
    withEnv(['CGO_ENABLED=0', 'GOOS=linux']) {
        sh 'go build -o app ./cmd/...'
    }

    // 归档制品
    archiveArtifacts artifacts: 'bin/*', fingerprint: true

    // 发布测试报告
    junit 'reports/*.xml'

    // 标记构建状态
    currentBuild.result = 'SUCCESS'
}
```

---

## 3 Go 项目 Jenkinsfile 实战

### 3.1 标准 Go CI Pipeline

```groovy
pipeline {
    agent {
        docker {
            image 'golang:1.24'
            args '-v go-mod-cache:/go/pkg/mod'  // 持久化模块缓存
        }
    }

    environment {
        CGO_ENABLED = '0'
        GOFLAGS     = '-trimpath'
        APP_NAME    = 'myapp'
    }

    options {
        timeout(time: 15, unit: 'MINUTES')  // 全局超时
        retry(1)                            // 失败不重试
        disableConcurrentBuilds()           // 禁止并行构建同一分支
        buildDiscarder(logRotator(
            numToKeepStr: '20',             // 保留最近 20 次构建
            artifactNumToKeepStr: '5'
        ))
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Lint') {
            steps {
                sh '''
                    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2
                    golangci-lint run --timeout=5m ./...
                '''
            }
        }

        stage('Test') {
            steps {
                sh '''
                    go test -race -coverprofile=coverage.out -covermode=atomic ./...
                    go tool cover -func=coverage.out
                '''
            }
            post {
                always {
                    // 保留覆盖率报告
                    archiveArtifacts artifacts: 'coverage.out', allowEmptyArchive: true
                }
            }
        }

        stage('Coverage Gate') {
            steps {
                sh '''
                    COV=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
                    echo "Coverage: ${COV}%"
                    awk -v cov="$COV" 'BEGIN { exit (cov+0 >= 80) ? 0 : 1 }' || \
                        (echo "ERROR: 覆盖率 ${COV}% 低于 80% 门禁" && exit 1)
                '''
            }
        }

        stage('Build') {
            steps {
                sh '''
                    VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
                    COMMIT=$(git rev-parse --short HEAD)
                    BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

                    mkdir -p bin
                    go build -ldflags="-s -w \
                        -X 'main.version=${VERSION}' \
                        -X 'main.commit=${COMMIT}' \
                        -X 'main.buildTime=${BUILD_TIME}'" \
                        -o bin/${APP_NAME} ./cmd/${APP_NAME}
                '''
                archiveArtifacts artifacts: 'bin/*', fingerprint: true
            }
        }
    }

    post {
        success {
            echo '✅ Pipeline 成功'
        }
        failure {
            echo '❌ Pipeline 失败'
            // 可集成通知：邮件、钉钉、企业微信
            // mail to: 'team@example.com', subject: "Build Failed: ${env.JOB_NAME}"
        }
        always {
            cleanWs()  // 清理工作空间
        }
    }
}
```

### 3.2 并行 Stage

无依赖的任务应并行执行，缩短反馈时间：

```groovy
stage('Quality') {
    parallel {
        stage('Lint') {
            steps {
                sh 'golangci-lint run --timeout=5m ./...'
            }
        }
        stage('Test') {
            steps {
                sh 'go test -race ./...'
            }
        }
        stage('Security') {
            steps {
                sh '''
                    go install github.com/securego/gosec/v2/cmd/gosec@v2.22.0
                    gosec ./...
                '''
            }
        }
    }
}
```

> **性能对比**: [performance/parallel-stages/](performance/parallel-stages/) — 串行 vs 并行 Stage 时间对比

### 3.3 Matrix 构建（多版本测试）

```groovy
stage('Test Matrix') {
    matrix {
        axes {
            axis {
                name 'GO_VERSION'
                values '1.23', '1.24'
            }
            axis {
                name 'GOOS'
                values 'linux', 'darwin'
            }
        }
        stages {
            stage('Test') {
                agent {
                    docker { image "golang:${GO_VERSION}" }
                }
                steps {
                    sh 'go test -race ./...'
                }
            }
        }
    }
}
```

---

## 4 凭据与密钥管理

### 4.1 Jenkins Credentials Store

Jenkins 内置凭据管理，支持多种类型：

| 类型 | 用途 | 示例 |
|------|------|------|
| Username with password | Git、Registry 认证 | Docker Hub 账号 |
| Secret text | API Key、Token | Slack Webhook |
| Secret file | 配置文件 | kubeconfig |
| SSH Username with private key | Git 拉取 | Deploy Key |
| Certificate | TLS 证书 | 客户端证书 |

### 4.2 在 Pipeline 中使用凭据

```groovy
pipeline {
    agent any

    stages {
        stage('Deploy') {
            steps {
                // 方式一：withCredentials 块
                withCredentials([
                    usernamePassword(
                        credentialsId: 'docker-hub',
                        usernameVariable: 'DOCKER_USER',
                        passwordVariable: 'DOCKER_PASS'
                    )
                ]) {
                    sh '''
                        echo "$DOCKER_PASS" | docker login -u "$DOCKER_USER" --password-stdin
                        docker push myapp:latest
                    '''
                }

                // 方式二：Secret text
                withCredentials([string(credentialsId: 'slack-webhook', variable: 'WEBHOOK_URL')]) {
                    sh 'curl -X POST -d "{\\"text\\":\\"Deploy complete\\"}" $WEBHOOK_URL'
                }

                // 方式三：Secret file（如 kubeconfig）
                withCredentials([file(credentialsId: 'kubeconfig', variable: 'KUBECONFIG')]) {
                    sh 'kubectl apply -f k8s/'
                }
            }
        }
    }
}
```

### 4.3 凭据作用域

```
Jenkins 全局 → Folder 级别 → Job 级别
```

**最佳实践**：
- 按 Folder 隔离不同团队/项目的凭据
- 生产环境凭据只在 production Folder 可见
- 定期轮换凭据，启用审计日志

> **反例**: [trap/hardcoded-credentials/](trap/hardcoded-credentials/) — 密钥硬编码在 Jenkinsfile 中

---

## 5 Agent 与构建环境

### 5.1 Agent 类型

```groovy
// 任意可用节点
agent any

// 指定标签的节点
agent { label 'linux && go' }

// Docker 容器（推荐 Go 项目）
agent {
    docker {
        image 'golang:1.24'
        args '-v go-mod-cache:/go/pkg/mod'
    }
}

// Dockerfile 自定义镜像
agent {
    dockerfile {
        filename 'Dockerfile.ci'
        dir 'build'
        args '-v go-mod-cache:/go/pkg/mod'
    }
}

// Kubernetes Pod
agent {
    kubernetes {
        yaml '''
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: golang
    image: golang:1.24
    command: ['sleep', '99d']
    volumeMounts:
    - name: go-cache
      mountPath: /go/pkg/mod
  volumes:
  - name: go-cache
    persistentVolumeClaim:
      claimName: go-mod-cache
'''
    }
}
```

### 5.2 Go 项目推荐 Agent 策略

```groovy
pipeline {
    // 全局默认用 Docker 隔离
    agent {
        docker {
            image 'golang:1.24'
            args '''
                -v go-mod-cache:/go/pkg/mod
                -v go-build-cache:/root/.cache/go-build
            '''
        }
    }

    stages {
        stage('CI') {
            steps {
                sh 'make lint test build'
            }
        }

        // 部署阶段用特定节点（有 kubectl / docker push 权限）
        stage('Deploy') {
            agent { label 'deployer' }
            steps {
                sh 'kubectl apply -f k8s/'
            }
        }
    }
}
```

> **反例**: [trap/fat-agent/](trap/fat-agent/) — 所有工具装在一个大 Agent 上，不隔离

---

## 6 Shared Library（共享库）

### 6.1 为什么需要 Shared Library

当多个 Go 项目的 Jenkinsfile 高度相似时，应抽取公共逻辑到 Shared Library：

```
多个项目 Jenkinsfile 长这样：
  项目 A: lint → test → build → deploy
  项目 B: lint → test → build → deploy
  项目 C: lint → test → build → deploy

→ 抽取为 Shared Library，一行调用
```

### 6.2 Shared Library 结构

```
jenkins-shared-lib/
├── vars/                      # 全局变量（Pipeline 可直接调用）
│   ├── goPipeline.groovy      # Go 项目标准流水线
│   └── notifySlack.groovy     # Slack 通知
├── src/                       # Groovy 类
│   └── com/myorg/GoBuilder.groovy
└── resources/                 # 静态资源
    └── com/myorg/Jenkinsfile.template
```

### 6.3 实现标准 Go Pipeline

```groovy
// vars/goPipeline.groovy
def call(Map config = [:]) {
    def goVersion   = config.goVersion ?: '1.24'
    def appName     = config.appName ?: 'app'
    def coverageMin = config.coverageMin ?: 80

    pipeline {
        agent {
            docker {
                image "golang:${goVersion}"
                args '-v go-mod-cache:/go/pkg/mod'
            }
        }

        options {
            timeout(time: 15, unit: 'MINUTES')
            disableConcurrentBuilds()
        }

        stages {
            stage('Lint') {
                steps {
                    sh '''
                        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2
                        golangci-lint run --timeout=5m ./...
                    '''
                }
            }

            stage('Test') {
                steps {
                    sh "go test -race -coverprofile=coverage.out ./..."
                    sh """
                        COV=\$(go tool cover -func=coverage.out | grep total | awk '{print \$3}' | sed 's/%//')
                        echo "Coverage: \${COV}%"
                        awk -v cov="\$COV" 'BEGIN { exit (cov+0 >= ${coverageMin}) ? 0 : 1 }'
                    """
                }
            }

            stage('Build') {
                steps {
                    sh """
                        CGO_ENABLED=0 go build -trimpath \
                            -ldflags='-s -w' \
                            -o bin/${appName} ./cmd/${appName}
                    """
                    archiveArtifacts artifacts: 'bin/*', fingerprint: true
                }
            }
        }

        post {
            always { cleanWs() }
        }
    }
}
```

### 6.4 使用 Shared Library

```groovy
// 项目 Jenkinsfile — 仅一行
@Library('my-shared-lib') _

goPipeline(
    appName: 'user-service',
    goVersion: '1.24',
    coverageMin: 85
)
```

**配置 Shared Library**：
Jenkins 管理 → Configure System → Global Pipeline Libraries：
- Name: `my-shared-lib`
- Source: Git 仓库地址
- Default version: `main`

---

## 7 CD：部署策略

### 7.1 部署流水线设计

```
CI（自动）                               CD（人工审批）
┌──────┐   ┌──────┐   ┌───────┐        ┌──────────┐        ┌──────────┐
│ Lint │──▶│ Test │──▶│ Build │──▶ ✅ ──▶│ Deploy   │──▶ 🔒 ──▶│ Deploy   │
│      │   │      │   │       │        │ Staging  │        │ Prod     │
└──────┘   └──────┘   └───────┘        └──────────┘        └──────────┘
                                            │                    │
                                            ▼                    ▼
                                       Smoke Test          Health Check
```

### 7.2 Staging + 审批 + Production

```groovy
pipeline {
    agent any

    stages {
        stage('CI') {
            agent {
                docker { image 'golang:1.24' }
            }
            steps {
                sh 'make lint test build'
                stash includes: 'bin/*', name: 'binary'
            }
        }

        stage('Deploy Staging') {
            agent { label 'deployer' }
            steps {
                unstash 'binary'
                withCredentials([file(credentialsId: 'kubeconfig-staging', variable: 'KUBECONFIG')]) {
                    sh '''
                        kubectl set image deployment/myapp \
                            myapp=myregistry/myapp:${BUILD_NUMBER} \
                            --namespace=staging
                        kubectl rollout status deployment/myapp --namespace=staging --timeout=120s
                    '''
                }
            }
        }

        stage('Smoke Test') {
            steps {
                sh '''
                    # 等待服务就绪
                    for i in $(seq 1 30); do
                        curl -sf http://staging.internal/health && break
                        sleep 2
                    done
                    # 关键接口验证
                    curl -sf http://staging.internal/api/v1/status | jq .
                '''
            }
        }

        stage('Approval') {
            steps {
                input message: '确认部署到生产环境？',
                      ok: '部署',
                      submitter: 'admin,release-manager'
            }
        }

        stage('Deploy Production') {
            agent { label 'deployer' }
            steps {
                unstash 'binary'
                withCredentials([file(credentialsId: 'kubeconfig-prod', variable: 'KUBECONFIG')]) {
                    sh '''
                        kubectl set image deployment/myapp \
                            myapp=myregistry/myapp:${BUILD_NUMBER} \
                            --namespace=production
                        kubectl rollout status deployment/myapp --namespace=production --timeout=180s
                    '''
                }
            }
        }
    }

    post {
        failure {
            echo '❌ Pipeline 失败，请检查'
        }
        success {
            echo '✅ 部署完成'
        }
    }
}
```

### 7.3 滚动更新 vs 蓝绿部署 vs 金丝雀

| 策略 | 原理 | 风险 | 回滚速度 | 适用场景 |
|------|------|------|---------|---------|
| 滚动更新 | 逐步替换旧 Pod | 中 | 中（rollback） | 一般服务 |
| 蓝绿部署 | 两套完整环境切换 | 低 | **秒级**（切回旧环境） | 核心服务 |
| 金丝雀 | 先导入小比例流量 | **最低** | 快（缩回金丝雀） | 大流量服务 |

**蓝绿部署 Jenkins 实现**：

```groovy
stage('Blue-Green Deploy') {
    steps {
        script {
            def current = sh(script: "kubectl get svc myapp -o jsonpath='{.spec.selector.version}'", returnStdout: true).trim()
            def target = (current == 'blue') ? 'green' : 'blue'
            echo "当前: ${current}, 目标: ${target}"

            // 部署到目标环境
            sh "kubectl set image deployment/myapp-${target} myapp=myregistry/myapp:${BUILD_NUMBER}"
            sh "kubectl rollout status deployment/myapp-${target} --timeout=120s"

            // Smoke test 目标环境
            sh "curl -sf http://myapp-${target}.internal/health"

            // 切换流量
            sh "kubectl patch svc myapp -p '{\"spec\":{\"selector\":{\"version\":\"${target}\"}}}'"
            echo "流量已切换到 ${target}"
        }
    }
}
```

---

## 8 Webhook 与触发机制

### 8.1 触发方式对比

| 方式 | 配置 | 延迟 | 适用场景 |
|------|------|------|---------|
| **Webhook**（推荐） | Git 仓库配置 Hook URL | 秒级 | 主流方式 |
| 轮询 SCM | `pollSCM('H/5 * * * *')` | 最长 5 分钟 | 无法配 Webhook 时 |
| 定时构建 | `cron('0 2 * * *')` | 固定时间 | 夜间构建、定期回归 |
| 手动触发 | `parameters { }` | 即时 | 发布、回滚 |

### 8.2 GitHub Webhook 配置

```groovy
pipeline {
    triggers {
        // GitHub Webhook（需安装 GitHub plugin）
        githubPush()
    }
    // ...
}
```

GitHub 仓库设置：
1. Settings → Webhooks → Add webhook
2. Payload URL: `http://jenkins.example.com/github-webhook/`
3. Content type: `application/json`
4. Events: `Push` + `Pull Request`

### 8.3 带参数的手动触发

```groovy
pipeline {
    parameters {
        choice(name: 'ENVIRONMENT', choices: ['staging', 'production'], description: '部署环境')
        string(name: 'VERSION', defaultValue: '', description: '指定版本（留空用最新）')
        booleanParam(name: 'SKIP_TESTS', defaultValue: false, description: '跳过测试（紧急修复用）')
    }

    stages {
        stage('Test') {
            when { expression { !params.SKIP_TESTS } }
            steps {
                sh 'go test -race ./...'
            }
        }

        stage('Deploy') {
            steps {
                echo "Deploying to ${params.ENVIRONMENT}"
                sh "deploy.sh ${params.ENVIRONMENT} ${params.VERSION ?: env.BUILD_NUMBER}"
            }
        }
    }
}
```

---

## 9 进阶技巧

### 9.1 Pipeline 缓存策略

Go 模块缓存对 Jenkins 性能影响巨大：

```groovy
agent {
    docker {
        image 'golang:1.24'
        // 使用 Docker Named Volume 持久化缓存
        args '''
            -v go-mod-cache:/go/pkg/mod
            -v go-build-cache:/root/.cache/go-build
        '''
    }
}
```

> **性能对比**: [performance/cache-strategy/](performance/cache-strategy/) — 有缓存 vs 无缓存的构建时间对比

### 9.2 多分支 Pipeline（Multibranch）

Jenkins Multibranch Pipeline 自动为每个分支/PR 创建 Job：

```
配置：
  New Item → Multibranch Pipeline
  Branch Sources → Git / GitHub
  Scan Interval → 1 minute

效果：
  main    → 自动构建 + 部署
  develop → 自动构建
  feature/* → 自动构建（可选）
  PR #123 → 自动构建 + 状态回写
```

```groovy
// 根据分支决定行为
stage('Deploy') {
    when {
        branch 'main'  // 仅 main 分支执行
    }
    steps {
        sh 'make deploy'
    }
}
```

### 9.3 构建状态通知

```groovy
post {
    success {
        // 钉钉通知（需安装 DingTalk 插件）
        sh """
            curl -X POST 'https://oapi.dingtalk.com/robot/send?access_token=\${DINGTALK_TOKEN}' \
                -H 'Content-Type: application/json' \
                -d '{"msgtype":"text","text":{"content":"✅ ${env.JOB_NAME} #${env.BUILD_NUMBER} 构建成功"}}'
        """
    }
    failure {
        // 企业微信通知
        sh """
            curl -X POST '\${WECHAT_WEBHOOK}' \
                -H 'Content-Type: application/json' \
                -d '{"msgtype":"text","text":{"content":"❌ ${env.JOB_NAME} #${env.BUILD_NUMBER} 构建失败\\n${env.BUILD_URL}"}}'
        """
    }
}
```

### 9.4 Pipeline 调试技巧

```groovy
// 1. 打印环境变量
stage('Debug') {
    steps {
        sh 'env | sort'
        sh 'go env'
    }
}

// 2. Replay 功能
// Jenkins UI → 构建历史 → Replay → 修改 Jenkinsfile 临时调试

// 3. 使用 catchError 不中断后续 Stage
stage('Optional Lint') {
    steps {
        catchError(buildResult: 'UNSTABLE', stageResult: 'FAILURE') {
            sh 'golangci-lint run ./...'
        }
    }
}
```

### 9.5 资源锁与并发控制

```groovy
// 方式一：全局禁止并发
options {
    disableConcurrentBuilds()
}

// 方式二：Lockable Resource（需安装插件）
stage('Deploy') {
    options {
        lock('production-deploy')  // 同一时间只有一个 Job 能部署
    }
    steps {
        sh 'make deploy'
    }
}
```

---

## 总结

| 实践 | 关键点 |
|------|--------|
| Pipeline 语法 | 始终用 Declarative，复杂逻辑用 `script { }` 嵌入 |
| Agent | Docker 容器隔离，挂载缓存 Volume |
| 凭据 | `withCredentials` 块引用，禁止硬编码 |
| 缓存 | 必须挂载 go mod + build cache |
| 超时 | `timeout` + `retry` 防止无限挂起 |
| 并行 | 无依赖 Stage 用 `parallel` 并行执行 |
| CD | Staging → 审批 → Production，至少两个环境 |
| Shared Library | 多项目复用，一行 Jenkinsfile |

**常见陷阱**：
- 密钥硬编码在 Jenkinsfile：[trap/hardcoded-credentials/](trap/hardcoded-credentials/)
- 不清理工作空间：[trap/no-cleanup/](trap/no-cleanup/)
- 不设超时导致 Pipeline 挂起：[trap/no-timeout/](trap/no-timeout/)
- 所有工具装在一个大 Agent：[trap/fat-agent/](trap/fat-agent/)

**性能对比**：
- 串行 vs 并行 Stage：[performance/parallel-stages/](performance/parallel-stages/)
- 缓存 vs 无缓存构建：[performance/cache-strategy/](performance/cache-strategy/)