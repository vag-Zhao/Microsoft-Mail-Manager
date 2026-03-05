# 更新日志

本文件记录项目的重要变更，格式参考 Keep a Changelog，并按功能维度整理。

---

## [Unreleased] - 2026-03-05

### 对比原始版本（1.0.0）差异汇总
- **数据安全能力增强**：从原始版本的基础账号管理，升级为敏感字段加密存储（密码 / RefreshToken / AccessToken），并补充 HTML 内容清理策略。
- **导入链路升级**：账号导入由“仅返回成功数量”升级为结构化结果（`total/success/failed/errors`），可准确反馈失败原因。
- **导入写入策略优化**：数据库写入由覆盖式逻辑升级为 `INSERT ... ON CONFLICT`，减少历史字段被意外覆盖风险。
- **批量操作可靠性提升**：批量删除、批量移动等操作引入事务保护，避免部分成功、部分失败导致状态不一致。
- **邮件协议与稳定性提升**：在原先 REST 能力基础上完善 IMAP 协议流程、连接池与优雅关闭，降低连接残留与并发竞态风险。
- **Token 管理增强**：增加缓存失效与周期清理机制，配合自动刷新逻辑，提升长时间运行稳定性。
- **前端体验改进**：补充关于信息展示、右键菜单与交互细节优化，统一导入导出体验与状态反馈。
- **构建发布能力完善**：已验证 Windows 独立可执行文件可打包输出（`build/bin/邮箱管家.exe`）。

### 新增 Added
- 新增**数据安全模块**：
  - `internal/security/encryption.go`：基于 **AES-256-GCM** 的敏感字段加解密能力（本地密钥文件管理）。
  - `internal/security/sanitizer.go`：HTML 清理策略封装（Strict / UGC Policy）。
- 新增导入结果模型 `internal/models/errors.go`：支持返回导入总数、成功数、失败数及错误明细。
- 新增应用信息接口与前端“关于”弹窗：
  - 后端接口：`app.go`（`GetAppInfo`）
  - 前端展示：`frontend/src/App.vue`
  - 测试：`app_info_test.go`
- 新增单账号导出能力（右键菜单“导出此邮箱”）：`frontend/src/App.vue`
- 新增 IMAP 服务生命周期管理（后台清理协程、健康检查、优雅关闭 `Shutdown`）：`internal/services/imap_service.go`
- 新增测试：
  - `internal/services/imap_service_test.go`
  - `internal/security/sanitizer_test.go`

### 优化 Changed
- 账号导入流程升级为**结构化结果返回**（`total/success/failed`）：
  - 后端：`app.go`、`internal/services/account_service.go`
  - 前端：`frontend/src/stores/account.ts`
  - 模型：`internal/models/errors.go`
- 账号导入写入逻辑改为 **INSERT ... ON CONFLICT**（替代 `INSERT OR REPLACE`）：`internal/services/account_service.go`
- 批量删除与批量移动账号改为**事务化处理**并使用预处理语句：`app.go`
- Token 缓存机制增强（新增周期性过期缓存清理）：`app.go`
- `GraphService` / `TokenService` 引入带超时 HTTP Client：
  - `internal/services/graph_service.go`
  - `internal/services/token_service.go`
- 前端交互体验优化（右键菜单防溢出、子菜单定位、动画统一、iframe 滚动条处理、统计卡片调整）：`frontend/src/App.vue`
- 数据库连接池与迁移能力增强：`internal/database/sqlite.go`

### 修复 Fixed
- 修复应用关闭时 IMAP 连接可能残留的问题（退出时显式关闭连接池）：`app.go`、`internal/services/imap_service.go`
- 修复 IMAP 客户端并发命令潜在竞态（`command()` 增加互斥保护）：`internal/services/imap_service.go`
- 修复批量操作原子性问题（中途失败导致部分成功）：`app.go`
- 修复导入后前端结果反馈不完整问题（改为读取结构化结果）：`frontend/src/stores/account.ts`

### 安全 Security
- 敏感字段（密码、RefreshToken、AccessToken）支持加密存储与读取解密：
  - 业务层：`internal/services/account_service.go`
  - 加密实现：`internal/security/encryption.go`
- 数据库迁移新增 `encrypted` 标记列，并执行历史明文数据迁移加密：`internal/database/sqlite.go`
- 邮件 HTML 清理能力接入（清理入口统一）：`app.go`、`internal/security/sanitizer.go`

### 依赖 Dependencies
- 新增依赖：`github.com/microcosm-cc/bluemonday v1.0.27`（`go.mod`、`go.sum`）
- 间接新增：`github.com/aymerick/douceur`、`github.com/gorilla/css`（`go.mod`、`go.sum`）

### 移除 Removed
- 删除文档文件：`profile_readme.md`
- 前端移除“已售/未售”标记、筛选与批量标记逻辑：`frontend/src/App.vue`
- 后端精简大量 Graph/IMAP 逐步调试日志：
  - `internal/services/graph_service.go`
  - `internal/services/imap_service.go`

---

## [1.2.0] - 2026-01-16

### 新增
- IMAP 服务器自动选择（个人账户 `imap-mail.outlook.com`，企业账户 `outlook.office365.com`）
- 底部状态栏加载动画
- 协议更新事件监听（`protocol-updated`），实时更新账号协议标签
- 全链路调试日志（前端 / 后端）

### 优化
- 协议检测从邮箱后缀规则升级为基于 `account.Protocol` 字段判断
- REST API → IMAP 自动回退机制（O2 失败后自动尝试 IMAP 并标记协议）
- TLS 连接配置增强（SNI、30 秒超时、TLS 1.2+）
- 账号列表展示协议类型（IMAP / O2）
- 文件夹数量展示逻辑优化（优先 `totalItemCount`）

### 修复
- 深色模式下分组右键菜单悬停可读性问题
- 邮件详情加载动画边框样式兼容性问题
- 切换账号时请求中断处理，避免旧请求覆盖新数据

---

## [1.1.0] - 2026-01-16

### 新增
- Hotmail 邮箱 IMAP 协议支持（XOAUTH2）
- 账号级缓存机制
- 右键菜单“刷新邮件”
- 邮件详情 Loading 动画
- 点击账号自动进入收件箱

### 优化
- IMAP 性能优化（预编译正则）
- 仅对关键文件夹执行 STATUS 查询
- 连接池复用
- 前端文件夹数量展示优化
- TLS 配置优化

### 修复
- Outlook Token 刷新 scope 问题（`invalid_grant`）
- IMAP 文件夹名解析兼容性
- 文件夹 ID 映射问题
- HTML 邮件脚本清理导致的告警相关问题
- 清空分组后视图状态复位问题

---

## [1.0.0] - 2026-01-15

### 初始版本
- 多账号管理（导入、分组、删除）
- Outlook REST API 支持
- 邮件列表与详情查看
- 附件下载
- 深色模式
- Token 自动刷新
