# east-money

东方财富自动交易接口 Go 语言实现，基于 [emtl](https://github.com/riiy/emtl) 的Golang版本实现与优化。

[![Go Version](https://img.shields.io/badge/Go-1.26.1-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

## 特性

- **完整 API 覆盖**：登录、资产查询、委托/成交查询、买入/卖出、撤单、行情
- **Token 自动管理**：缓存优先 + 双检锁 + TTL 过期 + 自动重登
- **内置 OCR**：基于 [ddddocr](https://github.com/sml2h3/ddddocr) ONNX 模型本地识别验证码，通过 [go-ocr](https://github.com/muzimu/go-ocr) 调用
- **可插拔架构**：Cache / Recognizer / Logger 均通过接口注入，支持自定义实现
- **重试机制**：指数退避重试，应对网络抖动
- **CLI 工具**：完整的命令行界面，支持买卖查撤全流程

## 快速开始

### 环境要求

| 依赖 | 说明 | 安装方式 |
| ---- | ---- | -------- |
| Go | 1.26.1 | [go.dev/dl](https://go.dev/dl/) |
| ONNX Runtime + 模型 | OCR 推理引擎与模型文件 | 见下方 |

### 安装

```bash
go get github.com/muzimu/east-money
```

### 准备 OCR 环境

本项目使用 [go-ocr](https://github.com/muzimu/go-ocr) 进行验证码识别，需要下载模型文件及 ONNX Runtime 动态链接库。

#### 第一步：下载模型及字典

```bash
git clone https://github.com/muzimu/go-ocr-model.git
```

仓库包含：

- `ddddocr/common.onnx` — OCR 识别模型
- `ddddocr/dict.txt` — 字符集字典

#### 第二步：下载 ONNX Runtime 库

从 [ONNX Runtime Releases](https://github.com/microsoft/onnxruntime/releases) 下载对应平台的动态链接库，放入 `go-ocr-model/lib/` 目录：

| 平台 | 文件 | 说明 |
| ---- | ---- | ---- |
| macOS Apple Silicon | `onnxruntime_arm64.dylib` | 下载 `onnxruntime-osx-arm64-*.tgz` 提取 |
| macOS Intel | `onnxruntime_amd64.dylib` | 下载 `onnxruntime-osx-x86_64-*.tgz` 提取 |
| Linux AMD64 | `onnxruntime_amd64.so` | 下载 `onnxruntime-linux-x64-*.tgz` 提取 |
| Windows | `onnxruntime.dll` | 下载 `onnxruntime-win-x64-*.zip` 提取 |

### 作为库使用

提供两种验证码识别方式：**本地 OCR** 和 **远程 OCR**，按需选用。

#### 方式一：本地 OCR（内置 ONNX 模型）

适用于单机部署、低延迟场景。需要下载模型文件及 ONNX Runtime 库。

```go
package main

import (
    "fmt"
    "log"

    "github.com/muzimu/east-money/captcha"
    "github.com/muzimu/east-money/client"
)

func main() {
    // 直接创建本地 OCR 识别器
    recognizer, err := captcha.NewDefaultRecognizer(
        "./go-ocr-model/ddddocr/common.onnx",                 // 模型文件
        "./go-ocr-model/ddddocr/dict.txt",                    // 字符集
        "./go-ocr-model/lib/onnxruntime_arm64.dylib",         // macOS Apple Silicon
        // "./go-ocr-model/lib/onnxruntime_amd64.dylib",      // macOS Intel
        // "./go-ocr-model/lib/onnxruntime_amd64.so",         // Linux AMD64
        // "./go-ocr-model/lib/onnxruntime.dll",              // Windows
    )
    if err != nil {
        log.Fatal(err)
    }
    defer recognizer.Close()

    // 创建交易客户端
    c, err := client.NewClient("your-username", "your-password", recognizer)
    if err != nil {
        log.Fatal(err)
    }

    // 查询资产与持仓
    resp, err := c.QueryAssetAndPosition()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%+v\n", resp)
}
```

#### 方式二：远程 OCR（ddddocr-fastapi）

适用于容器化部署、无需本地 ONNX 模型的场景。需先启动 [ddddocr-fastapi](https://github.com/sml2h3/ddddocr-fastapi) 服务，然后直接使用 `NewClient` + `RemoteRecognizer`。

```bash
# 启动 ddddocr-fastapi 服务（默认监听 8000 端口）
docker run -d -p 8000:8000 sml2h3/ddddocr-fastapi
```

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/muzimu/east-money/captcha"
    "github.com/muzimu/east-money/client"
)

func main() {
    // 创建远程 OCR 识别器（默认连接 http://localhost:8000/ocr）
    recognizer := captcha.NewRemoteRecognizer(
        captcha.WithRemoteTimeout(5 * time.Second),
    )

    // 或指定自定义地址
    // recognizer := captcha.NewRemoteRecognizer(
    //     captcha.WithRemoteEndpoint("http://10.0.0.5:8000/ocr"),
    // )

    // 创建交易客户端
    c, err := client.NewClient("your-username", "your-password", recognizer)
    if err != nil {
        log.Fatal(err)
    }

    // 查询资产与持仓
    resp, err := c.QueryAssetAndPosition()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%+v\n", resp)
}
```

### 高级配置

```go
c, err := client.NewClient(
    "username", "password", recognizer,
    client.WithRetry(5, time.Second),         // 重试 5 次，基础间隔 1s
    client.WithDuration(60),                  // 会话保持 60 分钟
    client.WithLogger(myLogger),              // 注入自定义日志
    client.WithLoginFailureCallback(func(err error) {
        // 登录失败告警
    }),
)
```

### 使用 Redis 缓存

默认使用内存缓存。如需分布式部署，替换为 Redis：

```go
c, err := client.NewClient("username", "password", recognizer)
c.SetCache(redisCache) // 实现 cache.Cache 接口即可
```

### 自定义验证码识别器

实现 `captcha.Recognizer` 接口即可注入任意识别逻辑。内置的 `RemoteRecognizer` 也是基于此接口实现。

```go
type MyRecognizer struct{}

func (m *MyRecognizer) Recognize(img image.Image) (string, error) {
    // 将 img 发送到你的 OCR 服务，例如调用第三方 API
    return remoteOCR(img)
}
func (m *MyRecognizer) Close() error { return nil }

c, err := client.NewClient("username", "password", &MyRecognizer{})
```

如果只是调用一个 HTTP 远端的 OCR 服务，可直接使用内置的 `captcha.NewRemoteRecognizer`（见上方 [方式二](#方式二远程-ocr内置-http-调用)）。

## CLI 使用

### 配置文件

CLI 支持 YAML 配置文件，优先级：**CLI 参数 > 环境变量 > 配置文件 > 默认值**。

在项目根目录创建 `config.yaml`：

```yaml
# ./config.yaml
user: "your_account"
password: "your_password"

ocr:
  model: "./go-ocr/ddddocr/common.onnx"
  dict: "./go-ocr/ddddocr/dict.txt"
  onnx_lib: ""  # 留空自动检测当前平台
```

也可通过 `--config` 指定其他路径。

### 命令示例

```bash
# 编译 可在.目录下执行 make build
go build -o eastmoney ./cmd/eastmoney

# 使用配置文件（零参数）
./eastmoney login

# 或使用命令行参数
./eastmoney login -u your_account -p your_password

# 查询资产
./eastmoney query asset

# 查询当日委托
./eastmoney query order

# 查询当日成交
./eastmoney query trade

# 买入（代码-价格-数量）
./eastmoney buy 000001-10.50-100

# 卖出
./eastmoney sell 600519-1850.00-100

# 撤单（委托日期_委托编号）
./eastmoney cancel 20240520_130662

# 行情
./eastmoney price 000001
```

### 命令行参数

| 参数 | 简写 | 说明 | 默认值 |
| ---- | ---- | ---- | ------ |
| `--config` | — | 配置文件路径 | `./config.yaml` |
| `--user` | `-u` | 账户用户名 | `$EM_USERNAME` |
| `--pass` | `-p` | 账户密码 | `$EM_PASSWORD` |
| `--model` | — | ONNX 模型路径 | `./go-ocr/ddddocr/common.onnx` |
| `--onnx-lib` | — | ONNX Runtime 库路径 | 自动检测（`go-ocr/lib/`） |

## API 参考

### Client 方法

| 方法 | 说明 |
| ---- | ---- |
| `QueryAssetAndPosition()` | 查询账户资产与持仓 |
| `QueryOrders()` | 查询当日委托 |
| `QueryTrades()` | 查询当日成交 |
| `QueryHistoryOrders(params)` | 查询历史委托 |
| `QueryHistoryTrades(params)` | 查询历史成交 |
| `QueryFundsFlow(params)` | 查询资金流水 |
| `CreateOrder(req)` | 提交买入/卖出委托 |
| `CancelOrder(orderStr)` | 撤销委托 |
| `GetLastPrice(code, market)` | 查询股票最新价格（无需登录） |
| `GetValidateKey()` | 获取当前有效会话凭证 |
| `ForceReLogin()` | 强制重新登录 |
| `SetCache(cache)` | 替换缓存后端 |

### 配置选项

| 选项 | 说明 |
| ---- | ---- |
| `WithRetry(max, wait)` | 设置重试次数与基础等待时间 |
| `WithDuration(minutes)` | 登录会话时长（分钟） |
| `WithLogger(logger)` | 注入日志记录器 |
| `WithLoginFailureCallback(cb)` | 登录失败回调 |
| `WithCaptchaRecognizer(r)` | 自定义验证码识别器 |
| `WithHTTPClient(hc)` | 自定义 HTTP 客户端（测试用） |

## 架构

```text
├── cache/            ← 缓存接口 + 内存/文件实现（可替换为 Redis）
├── captcha/          ← 验证码识别接口 + 本地 ddddocr ONNX 实现 + 远程 HTTP 实现
├── client/           ← HTTP 客户端（登录、查询、交易）
├── credential/       ← 会话凭证管理（双检锁 + TTL）
├── const.go          ← API 端点常量、RSA 公钥
└── cmd/eastmoney/    ← CLI 工具（cobra）
```

## 与 Python 原版对比

| 能力 | emtl (Python) | east-money (Go) |
| ---- | :---: | :---: |
| 登录 | ✅ | ✅ |
| 资产持仓 | ✅ | ✅ |
| 委托查询 | ✅ | ✅ |
| 成交查询 | ✅ | ✅ |
| 历史查询 | ✅ | ✅ |
| 买入/卖出 | ✅ | ✅ |
| 撤单 | ✅ | ✅ |
| 行情 | ✅ | ✅ |
| Token 管理 | ❌ 全局变量 | ✅ 双检锁+TTL |
| 重试机制 | ❌ | ✅ 指数退避 |
| 可插拔缓存 | ❌ | ✅ |
| CLI | ⚠️ 仅参数解析 | ✅ 完整实现 |

## 运行测试

```bash
# 全部测试
go test ./...

# 跳过需模型的测试
go test -short ./...

# 单个包
go test -v ./captcha/...
```

## License

MIT
