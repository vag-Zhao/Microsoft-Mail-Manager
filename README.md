# 邮箱管家（Outlook Mail Manager）

基于 **Wails + Go + Vue 3** 的桌面邮件管理应用，面向 Outlook / Hotmail 多账号场景，提供账号导入导出、分组管理、邮件查看、协议自动切换与本地安全存储能力。

## 核心能力

### 1) 多账号与分组管理
- 批量导入账号（支持 `.zgsacc` 归档文件与 `.txt` 文本文件）
- 分组创建、删除、清空、账号移动
- 批量删除、批量移动（事务保护）
- 账号 Token 有效性检测

### 2) 邮件读取与查看
- 文件夹列表、分页邮件列表、邮件详情
- HTML 邮件安全渲染（脚本与危险协议清理）
- 附件列表读取与下载

### 3) 双协议访问策略
- 优先使用 Outlook REST API（O2）
- 失败时自动回退 IMAP，并记录账号协议类型
- 后续请求按协议类型直连，减少重复探测

### 4) 导入导出链路优化
- 导出使用自定义归档格式（`.zgsacc`），避免明文可直接读取
- 兼容外部明文 `.txt` 导入
- 导入结果结构化返回：`total / success / failed / errors`

### 5) 本地安全与稳定性
- 敏感字段（Password / RefreshToken / AccessToken）本地加密存储
- Token 缓存 + 过期清理机制
- IMAP 连接池、健康检查与优雅关闭
- SQLite 连接池配置与自动迁移

---

## 技术栈

- **Desktop**: Wails v2
- **Backend**: Go 1.25
- **Frontend**: Vue 3 + TypeScript + Pinia + TailwindCSS
- **Database**: SQLite
- **Mail**: Microsoft REST API + IMAP (XOAUTH2)

---

## 项目结构

```text
.
├─ app.go                         # Wails 绑定入口与核心 API
├─ main.go                        # 程序入口
├─ internal/
│  ├─ database/sqlite.go          # 数据库初始化与迁移
│  ├─ models/                     # 数据模型
│  │  ├─ account.go
│  │  ├─ errors.go                # 导入结果模型
│  │  └─ mail.go
│  ├─ security/                   # 加密与安全清理
│  │  ├─ encryption.go
│  │  └─ sanitizer.go
│  ├─ services/                   # 业务服务层
│  │  ├─ account_service.go
│  │  ├─ graph_service.go
│  │  ├─ group_service.go
│  │  ├─ imap_service.go
│  │  └─ token_service.go
│  └─ utils/
│     ├─ account_archive.go       # .zgsacc 编解码
│     └─ parser.go                # 文本账号解析
├─ frontend/
│  ├─ src/App.vue
│  ├─ src/stores/
│  │  ├─ account.ts
│  │  └─ mail.ts
│  └─ wailsjs/                    # Wails 生成绑定
└─ CHANGELOG.md
```

---

## 运行与开发

### 环境要求
- Go 1.25+
- Node.js 18+
- Wails CLI v2+
- Windows 10/11（开发与本地打包）

### 安装依赖
```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 前端依赖
cd frontend
npm install
```

### 开发模式
```bash
wails dev
```

### 运行测试
```bash
go test ./...
```

### 构建 Windows 可执行文件
```bash
wails build
```
输出目录：`build/bin/`（例如 `邮箱管家.exe`）

> 说明：macOS 版本需要在 macOS 环境构建（`wails build -platform darwin/arm64`）。

---

## 数据与安全说明

- 本地数据库：`~/.outlook-mail-manager/data.db`
- 本地密钥文件：`~/.outlook-mail-manager/.key`
- 首次运行会自动完成数据库初始化与迁移
- 历史明文字段会在迁移流程中转换为加密存储

> 请妥善备份本地密钥文件；密钥丢失会导致已加密数据无法解密。

---

## 导入数据格式

支持两类来源：

1. **归档文件**：`.zgsacc`（推荐）
2. **明文文本**：`.txt`（兼容外部来源）

文本支持如下字段格式：

```text
邮箱----密码----ClientID----RefreshToken----分组名
```

或 Tab 分隔：

```text
邮箱<TAB>密码<TAB>ClientID<TAB>RefreshToken<TAB>分组名
```

---

## 界面截图

### 1) 邮件查看

![邮件查看界面](screenshots/mail-view.png)

### 2) 账号管理

![账号管理界面](screenshots/manage-view.png)

### 3) 深色模式

![深色模式界面](screenshots/dark-mode.png)

---

## 版本记录

请查看 [`CHANGELOG.md`](CHANGELOG.md) 获取详细变更历史。

---

## License

[MIT](LICENSE)
