# Go 微信 SDK（基于 iLink 协议）包管理规范

> 版本：v1.0  
> 适用：Golang SDK 设计 / 微信协议接入 / iLink 协议实现  
> 目标：可维护、可扩展、工程化

---

# 一、设计原则

本 SDK 设计遵循以下核心原则：

- 单一入口（Client）
- 分层清晰（API / 协议 / 传输）
- 对外稳定，对内可重构（internal）
- 高扩展性（Option + Middleware）
- 强类型约束（types 分层）

---

# 二、推荐目录结构

wechat-ilink-sdk/
├── go.mod
├── README.md
├── LICENSE
├── client.go
├── options.go
├── config.go
├── errors.go
├── version.go
├── api/
├── types/
├── ilink/
├── internal/
├── event/
├── examples/
└── docs/

---

# 三、核心设计规范

## 1. Client 统一入口

所有功能必须通过 Client 暴露，禁止全局函数。

## 2. Service 分层

每个业务领域一个 Service，通过 Client 挂载。

## 3. Option 模式

用于配置扩展，避免参数爆炸。

## 4. types 分层

请求/响应结构统一放在 types/，避免循环依赖。

## 5. iLink 协议层

负责协议封包、解包、连接管理。

## 6. internal 目录

隐藏实现细节，禁止外部引用。

---

# 四、事件系统

支持回调机制，必须异步、不阻塞。

---

# 五、错误处理

统一 Error 结构，支持 errors.Is / errors.As。

---

# 六、Transport 抽象

支持 TCP / WebSocket / Mock。

---

# 七、中间件机制

支持日志、限流、重试、tracing。

---

# 八、版本管理

遵循语义化版本（SemVer）。

---

# 九、反模式

- 全局函数设计
- 不使用 internal
- 协议与业务耦合
- context 滥用

---

# 十、架构分层

Client → API → iLink → Transport

---

# 十一、扩展建议

增加 agent 模块支持任务调度与自动化。

---

# 总结

Go SDK 的核心是构建一个可扩展的客户端体系。
