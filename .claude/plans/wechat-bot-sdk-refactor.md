# WeChat Bot SDK 重构计划

## 目标

将 `weixin-sdk` 重构为符合 Go SDK 最佳实践的 `wechat-bot-sdk`。

## 当前问题

| 问题 | 当前状态 | 目标状态 |
|------|----------|----------|
| 模块名 | `github.com/the-yex/weixin-sdk` | `github.com/the-yex/wechat-bot-sdk` |
| 根包名 | `weixin` | `wechatbot` |
| 导入方式 | 需要显式别名 | 直接导入 `wechatbot` |
| README 安装命令 | 错误 (`weixin`) | 正确 (`wechat-bot-sdk`) |
| 子包命名 | `api`, `auth` 等通用名 | 更具体的命名 |
| 版本管理 | 无 | 语义化版本 |

## 重构步骤

### Phase 1: 模块名和根包名更改

#### 1.1 更新 go.mod
```go
module github.com/the-yex/wechat-bot-sdk
```

#### 1.2 更新根包名
- `client.go` - `package weixin` → `package wechatbot`
- `doc.go` - `package weixin` → `package wechatbot`
- `errors.go` - `package weixin` → `package wechatbot`
- `options.go` - `package weixin` → `package wechatbot`

#### 1.3 更新所有 import 路径
所有文件中的 import 从：
```go
github.com/the-yex/weixin-sdk/...
```
改为：
```go
github.com/the-yex/wechat-bot-sdk/...
```

影响文件：
- 根目录: `client.go`, `options.go`
- `auth/login.go`
- `cdn/upload.go`, `cdn/download.go`, `cdn/client_test.go`
- `middleware/chain.go`, `middleware/retry.go`
- `plugin/plugin.go`, `plugin/registry.go`
- `messaging/context.go`
- `examples/basic-bot/main.go`, `examples/ai-assistant/main.go`

## 用户选择

- **模块名**: `wechat-bot-sdk`
- **重构范围**: 完整重构（包含子包重命名）

## 子包重命名确认

| 旧名称 | 新名称 | 说明 |
|--------|--------|------|
| `api/` | `ilink/` | 更具体，表明是 iLink 协议 API |
| `cdn/` | `media/` | 更符合 SDK 使用场景 |
| `auth/` | `login/` | 更准确描述登录流程 |

#### 当前结构 → 新结构

```
当前:                    新:
├── api/              →  ├── ilink/          # iLink API 客户端
│   ├── client.go        │   ├── client.go
│   ├── types.go         │   ├── types.go
│   ├── errors.go        │   ├── errors.go
│   └── ...              │   └── ...
├── auth/             →  ├── login/          # 登录流程
│   ├── login.go         │   ├── flow.go
│   └── store.go         │   └── ...
├── cdn/              →  ├── media/          # CDN 媒体处理
│   ├── upload.go        │   ├── upload.go
│   └── download.go      │   └── download.go
├── middleware/       →  ├── middleware/     # 保持不变
├── plugin/           →  ├── plugin/         # 保持不变
├── messaging/        →  (合并到根包)        # 简化
└── examples/         →  └── examples/       # 保持不变
```

**子包重命名理由：**
- `api/` → `ilink/`: 更具体，表明是 iLink 协议 API
- `auth/` → `login/`: 更准确描述登录流程
- `cdn/` → `media/`: 更符合 SDK 使用场景

### Phase 3: 导出类型优化

#### 3.1 根包导出常用类型
让用户可以直接从根包导入常用类型：

```go
// 根包导出
type Message = ilink.Message
type MessageType = ilink.MessageType
type LoginResult = ilink.LoginResult
```

#### 3.2 改进后的用户体验
```go
import "github.com/the-yex/wechat-bot-sdk"

// 之前需要
// import weixin "github.com/the-yex/weixin-sdk"
// import "github.com/the-yex/weixin-sdk/api"

func main() {
    client, _ := wechatbot.NewClient(wechatbot.WithToken("xxx"))
    client.Run(ctx, func(ctx context.Context, msg *wechatbot.Message) error {
        return client.SendText(ctx, msg.FromUserID, "Hello!")
    })
}
```

### Phase 4: 版本管理

#### 4.1 添加版本文件
创建 `version.go`:
```go
package wechatbot

const (
    Version = "1.0.0"
)
```

#### 4.2 API User-Agent 更新
```go
UserAgent = "wechat-bot-sdk-go/" + Version
```

### Phase 5: 文档更新

#### 5.1 README.md
- 更新安装命令
- 更新所有代码示例
- 更新 badge URL

#### 5.2 doc.go
更新包文档

#### 5.3 示例代码
更新 examples 目录下的代码

## 文件变更清单

### 必须修改的文件（Phase 1）

| 文件 | 变更内容 |
|------|----------|
| `go.mod` | 模块名 |
| `client.go` | 包名 + imports |
| `doc.go` | 包名 |
| `errors.go` | 包名 |
| `options.go` | 包名 + imports |
| `auth/login.go` | imports |
| `auth/store.go` | imports |
| `cdn/upload.go` | imports |
| `cdn/download.go` | imports |
| `cdn/crypto.go` | imports |
| `cdn/client_test.go` | imports |
| `middleware/chain.go` | imports |
| `middleware/logging.go` | imports |
| `middleware/recovery.go` | imports |
| `middleware/retry.go` | imports |
| `plugin/plugin.go` | imports |
| `plugin/registry.go` | imports |
| `messaging/context.go` | imports |
| `examples/basic-bot/main.go` | imports |
| `examples/ai-assistant/main.go` | imports |
| `README.md` | 安装命令 + 示例 |

### 新增文件

| 文件 | 说明 |
|------|------|
| `version.go` | 版本常量 |

## 风险评估

- **低风险**: 模块名和包名更改是纯文本替换
- **建议**: 先创建新分支进行重构，测试通过后合并

## 验证步骤

1. `go mod tidy`
2. `go build ./...`
3. `go test ./...`
4. 运行 examples 验证功能正常