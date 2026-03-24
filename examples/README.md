# QR Code Login Examples

本目录包含微信 iLink SDK 的扫码登录示例。

## 示例列表

### 1. simple-login

最简单的扫码登录示例，仅展示登录流程。

**运行方式：**
```bash
go run ./examples/simple-login/main.go
```

**功能：**
- 创建客户端（无 token）
- 生成并显示二维码
- 等待用户扫码确认
- 输出登录结果

### 2. qrcode-login

完整的扫码登录示例，展示登录流程和 Token 存储。

**运行方式：**
```bash
go run ./examples/qrcode-login/main.go
```

**功能：**
- 创建客户端和文件 Token 存储
- 生成并显示二维码（URL 和内容）
- 等待用户扫码确认
- 输出登录结果和账户信息
- 演示 Token 的保存和加载

### 3. qrcode-login-with-image

扫码登录 + 自动回复机器人示例。

**运行方式：**
```bash
go run ./examples/qrcode-login-with-image/main.go
```

**功能：**
- 包含 qrcode-login 的所有功能
- 启动消息监听循环
- 自动回复收到的文本消息
- 支持 Ctrl+C 退出

### 4. basic-bot (主目录)

完整的机器人示例，包含中间件、插件等高级功能。

**运行方式：**
```bash
go run ./examples/basic-bot/main.go
```

## 登录流程说明

1. **创建客户端** - 使用 `NewClient()` 创建 SDK 客户端
2. **检查 Token** - 检查是否有存储的 Token 可以直接登录
3. **生成二维码** - 调用 `Login()` 方法，SDK 会生成二维码
4. **显示二维码** - 在回调函数中获取二维码 URL 或内容
5. **用户扫码** - 用户使用微信扫描二维码并确认
6. **获取结果** - 登录成功后获取 AccountID、UserID 和 Token
7. **存储 Token** - 使用 TokenStore 保存 Token 以便下次使用

## Token 存储

SDK 提供两种 Token 存储方式：

### MemoryTokenStore（默认）
```go
tokenStore := login.NewMemoryTokenStore()
```

### FileTokenStore（推荐）
```go
tokenStore, err := login.NewFileTokenStore("")
if err != nil {
    // handle error
}
client, err := ilinksdk.NewClient(
    ilinksdk.WithTokenStore(tokenStore),
)
```

文件存储会将 Token 保存在 `~/.wechat-ilink/tokens.json`。

## 常见问题

**Q: 二维码过期怎么办？**
A: 二维码有效期为 5 分钟，过期后需要重新调用 Login() 方法生成新的二维码。

**Q: 如何自定义二维码显示？**
A: 在 Login 回调函数中，你可以：
- 打印 URL 让用户在浏览器打开
- 使用 qrcode 库生成终端二维码
- 生成图片文件供用户扫描
- 在 Web 界面显示二维码图片

**Q: 如何处理登录失败？**
A: Login() 方法返回 error，检查错误类型并重试或提示用户。
