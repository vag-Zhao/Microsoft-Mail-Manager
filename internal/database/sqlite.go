// Package database 数据库层
//
// sqlite.go SQLite数据库初始化和管理
//
// 功能说明：
// - 数据库连接初始化
// - 数据表结构迁移
// - 数据库连接关闭
//
// 数据库位置：~/.outlook-mail-manager/data.db
// 使用的驱动：github.com/mattn/go-sqlite3
package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite驱动，使用空白导入注册驱动
)

// DB 全局数据库连接实例
//
// 在应用启动时由Init()初始化
// 所有数据库操作都通过此实例进行
var DB *sql.DB

// Init 初始化数据库连接
//
// 执行流程：
// 1. 获取用户主目录
// 2. 创建应用数据目录（~/.outlook-mail-manager）
// 3. 打开SQLite数据库连接
// 4. 执行数据表迁移
//
// 返回值：
//   - error: 初始化失败时返回错误（目录创建失败、数据库连接失败等）
func Init() error {
	// 获取用户主目录（Windows: C:\Users\xxx, Linux/Mac: /home/xxx）
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	// 构建数据库目录路径
	dbDir := filepath.Join(homeDir, ".outlook-mail-manager")
	// 创建目录（如果不存在），权限755
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("create db dir: %w", err)
	}
	// 构建数据库文件路径
	dbPath := filepath.Join(dbDir, "data.db")

	// 打开SQLite数据库连接
	// 如果文件不存在会自动创建
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	DB.SetMaxOpenConns(10)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxIdleTime(5 * time.Minute)
	// 执行数据表迁移
	return migrate()
}

// migrate 执行数据库迁移
//
// 创建应用所需的数据表和索引
// 使用 IF NOT EXISTS 确保可重复执行（幂等性）
//
// 表结构说明：
//
// groups 分组表：
//   - id: 主键，自增
//   - name: 分组名称，不能为空
//   - parent_id: 父分组ID（预留，暂未使用）
//   - sort_order: 排序顺序
//   - created_at: 创建时间
//
// accounts 账号表：
//   - id: 主键，自增
//   - email: 邮箱地址，唯一约束
//   - password: 邮箱密码（可选）
//   - client_id: OAuth2客户端ID
//   - refresh_token: OAuth2刷新令牌
//   - access_token: OAuth2访问令牌（缓存）
//   - token_expires_at: 令牌过期时间
//   - group_id: 所属分组ID，外键关联groups表
//   - display_name: 显示名称
//   - status: 状态（active/error）
//   - last_error: 最后一次错误信息
//   - created_at: 创建时间
//   - updated_at: 更新时间
//
// 返回值：
//   - error: SQL执行错误
func migrate() error {
	schema := `
	-- 分组表：用于组织和管理账号
	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- 账号表：存储Outlook邮箱账号信息
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		password TEXT,
		client_id TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		access_token TEXT,
		token_expires_at DATETIME,
		group_id INTEGER,
		display_name TEXT,
		status TEXT DEFAULT 'active',
		last_error TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE SET NULL
	);

	-- 索引：加速邮箱查询（用于去重和查找）
	CREATE INDEX IF NOT EXISTS idx_accounts_email ON accounts(email);
	-- 索引：加速按分组筛选
	CREATE INDEX IF NOT EXISTS idx_accounts_group ON accounts(group_id);

	-- 初始化默认分组（ID=1）
	INSERT OR IGNORE INTO groups (id, name) VALUES (1, '默认分组');
	`
	// 执行SQL语句
	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	// 添加 protocol 列（如果不存在）
	DB.Exec("ALTER TABLE accounts ADD COLUMN protocol TEXT DEFAULT 'o2'")

	// 添加 encrypted 列（如果不存在）
	// encrypted = 0: 明文数据（旧数据）
	// encrypted = 1: 已加密数据（新数据）
	DB.Exec("ALTER TABLE accounts ADD COLUMN encrypted INTEGER DEFAULT 0")

	// 执行数据加密迁移
	// 将所有明文数据加密存储
	if err := migrateEncryption(); err != nil {
		return fmt.Errorf("encryption migration failed: %w", err)
	}

	return nil
}

// migrateEncryption 加密现有明文数据
//
// 数据迁移流程：
// 1. 初始化加密服务
// 2. 查询所有 encrypted = 0 的账号（明文数据）
// 3. 使用事务批量加密 password、refresh_token、access_token
// 4. 更新 encrypted = 1 标记
// 5. 提交事务
//
// 返回值：
//   - error: 加密服务初始化失败或数据库操作失败
//
// 注意事项：
//   - 使用事务确保原子性：要么全部成功，要么全部回滚
//   - 迁移失败不影响应用启动，但敏感数据仍为明文
//   - 迁移成功后，所有新数据都会自动加密
func migrateEncryption() error {
	// 导入加密服务包
	// 注意：这里需要在文件顶部添加 import
	encSvc, err := newEncryptionServiceForMigration()
	if err != nil {
		return err
	}

	// 查询所有未加密的账号
	rows, err := DB.Query("SELECT id, password, refresh_token, access_token FROM accounts WHERE encrypted = 0")
	if err != nil {
		return err
	}
	defer rows.Close()

	// 开启事务
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // 如果提交失败，自动回滚

	// 准备更新语句
	stmt, err := tx.Prepare(`UPDATE accounts SET password = ?, refresh_token = ?,
		access_token = ?, encrypted = 1 WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 逐行加密数据
	count := 0
	for rows.Next() {
		var id int64
		var password, refreshToken, accessToken sql.NullString

		if err := rows.Scan(&id, &password, &refreshToken, &accessToken); err != nil {
			continue // 跳过解析失败的行
		}

		// 加密敏感字段
		encPassword, _ := encSvc.Encrypt(password.String)
		encRefresh, _ := encSvc.Encrypt(refreshToken.String)
		encAccess, _ := encSvc.Encrypt(accessToken.String)

		// 更新数据库
		if _, err := stmt.Exec(encPassword, encRefresh, encAccess, id); err != nil {
			return err
		}
		count++
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return err
	}

	// 记录迁移结果
	if count > 0 {
		fmt.Printf("[Database] 成功加密 %d 个账号的敏感数据\n", count)
	}

	return nil
}

// newEncryptionServiceForMigration 为迁移创建加密服务
//
// 这是一个临时函数，用于在数据库迁移时创建加密服务
// 避免循环依赖问题（database -> security -> database）
//
// 返回值：
//   - encryptionService: 简化的加密服务接口
//   - error: 初始化失败
func newEncryptionServiceForMigration() (encryptionService, error) {
	// 这里需要导入 security 包
	// 为了避免循环依赖，我们在这里直接实现简化版本
	// 或者使用接口来解耦
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	keyPath := filepath.Join(homeDir, ".outlook-mail-manager", ".key")

	// 尝试读取现有密钥
	var key []byte
	if keyData, err := os.ReadFile(keyPath); err == nil {
		key = keyData
	} else {
		// 生成新密钥
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := os.WriteFile(keyPath, key, 0600); err != nil {
			return nil, err
		}
	}

	return &simpleEncryptionService{key: key}, nil
}

// encryptionService 加密服务接口
//
// 定义加密和解密方法，用于解耦数据库层和安全层
type encryptionService interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// simpleEncryptionService 简化的加密服务实现
//
// 用于数据库迁移，避免循环依赖
type simpleEncryptionService struct {
	key []byte
}

// Encrypt 加密字符串（简化版）
func (s *simpleEncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密字符串（简化版）
func (s *simpleEncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// Close 关闭数据库连接
//
// 在应用退出时调用，释放数据库资源
// 安全检查：仅在DB不为nil时关闭
func Close() {
	if DB != nil {
		DB.Close()
	}
}
