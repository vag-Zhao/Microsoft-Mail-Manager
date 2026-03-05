# 邮箱管家（Outlook Mail Manager）

基于 **Wails + Go + Vue 3** 的桌面邮件管理应用，面向 Outlook / Hotmail 多账号场景，提供账号导入导出、分组管理、邮件查看、协议自动切换与本地安全存储能力。

## 核心能力

<table>
  <tr>
    <td valign="top" width="50%">
      <strong>多账号管理</strong><br />
      导入/导出、分组管理、批量移动与删除。<br /><br />
      <strong>邮件查看</strong><br />
      文件夹浏览、分页列表、详情与附件下载。
    </td>
    <td valign="top" width="50%">
      <strong>协议自动切换</strong><br />
      优先 REST API，失败自动回退 IMAP 并持久化策略。<br /><br />
      <strong>本地安全存储</strong><br />
      敏感字段加密、Token 管理、连接池与迁移保障。
    </td>
  </tr>
</table>

---

## 界面截图

<table>
  <tr>
    <td align="center">
      <strong>邮件查看</strong><br />
      <img src="screenshots/mail-view.png" alt="邮件查看界面" width="100%" />
    </td>
    <td align="center">
      <strong>账号管理</strong><br />
      <img src="screenshots/manage-view.png" alt="账号管理界面" width="100%" />
    </td>
  </tr>
  <tr>
    <td align="center" colspan="2">
      <strong>深色模式</strong><br />
      <img src="screenshots/dark-mode.png" alt="深色模式界面" width="70%" />
    </td>
  </tr>
</table>

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

## 版本记录

请查看 [`CHANGELOG.md`](CHANGELOG.md) 获取详细变更历史。

---

## License

[MIT](LICENSE)
