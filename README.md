# east-money

东方财富自动交易接口 Go 语言实现，基于 [emtl](https://github.com/riiy/emtl) 的 Golang 版本实现与优化。

[![Go Version](https://img.shields.io/badge/Go-1.26.1-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](./LICENSE)

## 特性

- **完整 API 覆盖**：登录、资产查询、委托/成交查询、历史查询、资金流水、买入/卖出、撤单、行情
- **Token 自动管理**：缓存优先 + 双检锁 + TTL 过期 + 自动重登
- **双 OCR 模式**：本地 [ddddocr](https://github.com/sml2h3/ddddocr) ONNX 识别，或远程 [ddddocr-fastapi](https://github.com/sml2h3/ddddocr-fastapi) 服务识别
- **可插拔架构**：Cache / Recognizer / Logger 均通过接口注入，支持自定义实现
- **重试机制**：指数退避重试，应对网络抖动
- **CLI 工具**：完整命令行界面，支持买卖查撤全流程，并可在本地 OCR 与远程 OCR 间切换

## 快速开始

### Release 下载

可直接从 [Releases](https://github.com/muzimu/east-money/releases) 下载最新版本的二进制文件，打包了对应的模型文件、字典文件及 ONNX Runtime 动态链接库。

### 开发环境

| 依赖 | 说明 | 安装方式 |
| ---- | ---- | -------- |
| Go | 1.26.1 | [go.dev/dl](https://go.dev/dl/) |
| OCR | 本地 ONNX 或远程 HTTP 二选一 | 见下方 |

### 安装及构建

作为 Go 库使用：

```bash
go get github.com/muzimu/east-money
```

构建 CLI：

```bash
make build
# 或
go build -o eastmoney ./cmd/eastmoney
```

### 选择 OCR 模式

登录东方财富时需要识别验证码。本项目同时支持两种 OCR 模式：

| 模式 | 适用场景 | 依赖 | CLI 开启方式 |
| ---- | -------- | ---- | ------------ |
| 本地 OCR | 单机部署、低延迟、无需额外服务 | ddddocr 模型 + ONNX Runtime 动态库 | 默认模式；配置 `ocr.model` / `ocr.dict` / `ocr.onnx_lib` |
| 远程 OCR | 容器化、多实例共享 OCR、避免本机安装 ONNX Runtime | 兼容 `ddddocr-fastapi` 的 HTTP 服务 | 配置 `ocr.remote` 或传入 `--ocr-remote` |

> 远程 OCR 优先级高于本地 OCR ：只要设置了 `ocr.remote` 或 `--ocr-remote`，CLI 会跳过本地 ONNX 模型，直接调用远程服务。

## 准备 OCR 环境

### 方式一：本地 OCR

本地 OCR 通过 [go-ocr](https://github.com/muzimu/go-ocr) 调用 ddddocr ONNX 模型，需要准备模型、字典和 ONNX Runtime 动态链接库。

#### 1. 下载模型及字典

```bash
git clone https://github.com/muzimu/go-ocr-model.git
```

仓库包含：

- `ddddocr/common.onnx` — OCR 识别模型
- `ddddocr/dict.txt` — 字符集字典

#### 2. 下载 ONNX Runtime 库

从 [ONNX Runtime Releases](https://github.com/microsoft/onnxruntime/releases) 下载对应平台的动态链接库，放入 `go-ocr-model/lib/` 目录：

| 平台 | 文件 | 说明 |
| ---- | ---- | ---- |
| macOS Apple Silicon | `onnxruntime_arm64.dylib` | 下载 `onnxruntime-osx-arm64-*.tgz` 后提取 |
| macOS Intel | `onnxruntime_amd64.dylib` | 下载 `onnxruntime-osx-x86_64-*.tgz` 后提取 |
| Linux ARM64 | `onnxruntime_arm64.so` | 下载 `onnxruntime-linux-aarch64-*.tgz` 后提取 |
| Linux AMD64 | `onnxruntime_amd64.so` | 下载 `onnxruntime-linux-x64-*.tgz` 后提取 |
| Windows | `onnxruntime.dll` | 下载 `onnxruntime-win-x64-*.zip` 后提取 |

> CLI 的 `ocr.onnx_lib` 留空时会尝试自动检测系统路径；为了部署稳定，建议在本地 OCR 模式下显式填写动态库路径。

### 方式二：远程 OCR（ddddocr-fastapi）

远程 OCR 兼容 [ddddocr-fastapi](https://github.com/sml2h3/ddddocr-fastapi) 的 `POST /ocr` 接口：

- 请求：默认使用 `multipart/form-data`，字段名为 `file`
- 兼容：库层也支持 base64 `image` 字段
- 响应：`{"code": 200, "message": "Success", "data": "识别结果"}`

启动服务示例：

```bash
# 默认监听 8000 端口，OCR 接口为 http://localhost:8000/ocr
docker run -d -p 8000:8000 sml2h3/ddddocr-fastapi
```

CLI 使用远程 OCR：

配置文件：

```yaml
ocr:
  remote: "http://localhost:8000/ocr"
```

或命令行传入参数

```bash
./eastmoney login --ocr-remote http://localhost:8000/ocr
```

## 作为库使用

库层通过 `captcha.Recognizer` 接口接入验证码识别器。你可以直接使用内置的本地 OCR、远程 OCR，也可以实现自己的识别器。

### 本地 OCR（ONNX 模型）

```go
package main

import (
    "fmt"
    "log"

    "github.com/muzimu/east-money/captcha"
    "github.com/muzimu/east-money/client"
)

func main() {
    recognizer, err := captcha.NewDefaultRecognizer(
        "./go-ocr-model/ddddocr/common.onnx",
        "./go-ocr-model/ddddocr/dict.txt",
        "./go-ocr-model/lib/onnxruntime_arm64.dylib",
    )
    if err != nil {
        log.Fatal(err)
    }
    defer recognizer.Close()

    c, err := client.NewClient("your-username", "your-password", recognizer)
    if err != nil {
        log.Fatal(err)
    }

    resp, err := c.QueryAssetAndPosition()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%+v\n", resp)
}
```

### 远程 OCR（ddddocr-fastapi）

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
    recognizer := captcha.NewRemoteRecognizer(
        captcha.WithRemoteEndpoint("http://localhost:8000/ocr"),
        captcha.WithRemoteTimeout(5*time.Second),
    )
    defer recognizer.Close()

    c, err := client.NewClient("your-username", "your-password", recognizer)
    if err != nil {
        log.Fatal(err)
    }

    resp, err := c.QueryAssetAndPosition()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%+v\n", resp)
}
```

如果你的远程 OCR 服务使用 base64 `image` 字段，可开启 base64 传输：

```go
recognizer := captcha.NewRemoteRecognizer(
    captcha.WithRemoteEndpoint("http://localhost:8000/ocr"),
    captcha.WithRemoteBase64(),
)
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

实现 `captcha.Recognizer` 接口即可注入任意识别逻辑。内置的本地 OCR 与远程 OCR 也都基于该接口实现。

```go
type MyRecognizer struct{}

func (m *MyRecognizer) Recognize(img image.Image) (string, error) {
    return remoteOCR(img)
}

func (m *MyRecognizer) Close() error { return nil }

c, err := client.NewClient("username", "password", &MyRecognizer{})
```

## CLI 使用

### 配置文件

CLI 支持 YAML 配置文件，默认读取 `./config.yaml`，也可通过 `--config` 指定其他路径。

配置优先级：**CLI 参数 > 环境变量 > 配置文件 > 默认值**。

#### 本地 OCR 配置

```yaml
# ./config.yaml
user: "your_account"
password: "your_password"
log: ".eastmoney/eastmoney.log"

ocr:
  model: "./go-ocr-model/ddddocr/common.onnx"
  dict: "./go-ocr-model/ddddocr/dict.txt"
  # 建议显式填写 go-ocr-model/lib 下的对应文件，留空时 CLI 会尝试自动检测系统路径
  onnx_lib: "./go-ocr-model/lib/onnxruntime_arm64.dylib"
```

#### 远程 OCR 配置

```yaml
# ./config.yaml
user: "your_account"
password: "your_password"
log: ".eastmoney/eastmoney.log"

ocr:
  # 设置 remote 后，CLI 将跳过本地 model/dict/onnx_lib
  remote: "http://localhost:8000/ocr"
```

完整示例可参考 [`config.example.yaml`](./config.example.yaml)。

### 命令示例

```bash
# 编译
make build

# 使用配置文件登录
./eastmoney login

# 使用命令行参数登录
./eastmoney login -u your_account -p your_password

# 使用远程 OCR 登录（会覆盖配置文件中的 ocr.remote）
./eastmoney login --ocr-remote http://localhost:8000/ocr

# 指定本地 OCR 资源登录
./eastmoney login \
  --model ./go-ocr-model/ddddocr/common.onnx \
  --dict ./go-ocr-model/ddddocr/dict.txt \
  --onnx-lib ./go-ocr-model/lib/onnxruntime_arm64.dylib

# 查询资产
./eastmoney query asset

# 查询当日委托
./eastmoney query order

# 查询当日成交
./eastmoney query trade

# 查询当日可撤单委托
./eastmoney query revocable

# 查询可操作数量（逆回购价格可选 不影响查询结果）
./eastmoney query operate-amount 204001
./eastmoney query operate-amount 204001-1.700

# 查询可操作数量（股票及ETF价格必填）
./eastmoney query operate-amount 600519-1850

# 查询历史委托 / 成交 / 资金流水
./eastmoney query history-order --start 2026-01-01 --end 2026-01-31
./eastmoney query history-trade --start 2026-01-01 --end 2026-01-31
./eastmoney query funds --start 2026-01-01 --end 2026-01-31

# 买入（代码-价格-数量）
./eastmoney buy 000001-10.50-100

# 买入逆回购（自动识别交易类型）
./eastmoney buy 204001-1.415-10

# 卖出
./eastmoney sell 600519-1850.00-100

# 撤单（委托日期_委托编号 或 委托编号，自动补全市场/买卖标志）
./eastmoney cancel 20260714_1771064
./eastmoney cancel 1771064

# 行情（无需登录，无需 OCR）
./eastmoney price 000001

# JSON 输出
./eastmoney query asset --format json
```

### 命令行参数

| 参数 | 简写 | 说明 | 默认值 |
| ---- | ---- | ---- | ------ |
| `--config` | — | 配置文件路径 | `./config.yaml` |
| `--user` | `-u` | 账户用户名，也可用 `EM_USERNAME` | 空 |
| `--pass` | `-p` | 账户密码，也可用 `EM_PASSWORD` | 空 |
| `--model` | — | ddddocr 模型文件路径 | `./go-ocr-model/ddddocr/common.onnx` |
| `--dict` | — | ddddocr 字典文件路径 | `./go-ocr-model/ddddocr/dict.txt` |
| `--onnx-lib` | — | ONNX Runtime 共享库路径；留空时自动检测 | 自动检测 |
| `--ocr-remote` | — | 远程 OCR 服务地址；设置后跳过本地 ONNX 模型 | 空 |
| `--session` | — | 会话持久化文件路径 | `.eastmoney/session.json` |
| `--log` | — | 日志文件路径 | `.eastmoney/eastmoney.log` |
| `--format` | — | 输出格式：`json` / `human` | `human` |
| `--size` | — | 历史查询条数 | `20` |
| `--start` | — | 历史查询起始日期，格式 `2006-01-02` | 空 |
| `--end` | — | 历史查询结束日期，格式 `2006-01-02` | 空 |

## API 参考

### Client 方法

| 方法 | 说明 |
| ---- | ---- |
| `QueryAssetAndPosition()` | 查询账户资产与持仓 |
| `QueryOperateAmount(stockCode, price, tradeType)` | 查询指定证券的可操作数量 |
| `QueryOrders()` | 查询当日委托 |
| `QueryRevocableOrders()` | 查询当日可撤单委托 |
| `QueryTrades()` | 查询当日成交 |
| `QueryHistoryOrders(params)` | 查询历史委托 |
| `QueryHistoryTrades(params)` | 查询历史成交 |
| `QueryFundsFlow(params)` | 查询资金流水 |
| `CreateOrder(req)` | 提交买入/卖出委托 |
| `CancelOrder(req)` | 撤销委托（需提供委托日期、编号、市场、买卖标志） |
| `CancelOrderByID(orderStr)` | 按委托标识撤销委托，自动查询并补全市场/买卖标志 |
| `GetLastPrice(symbolCode)` | 查询股票最新价格（无需登录） |
| `GetSnapshot(symbolCode)` | 查询股票完整行情快照（无需登录） |
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
└── cmd/eastmoney/    ← CLI 工具（cobra），支持本地/远程 OCR 配置
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
| OCR 部署 | 本地脚本 | ✅ 本地 ONNX / 远程 HTTP |

## 运行测试

```bash
# 全部测试
go test ./...

# 跳过需模型或外部服务的测试
go test -short ./...

# 单个包
go test -v ./captcha/...
```

## License

MIT
