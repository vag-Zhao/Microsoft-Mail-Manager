// Package security 安全相关功能
//
// encryption.go 提供数据加密和解密服务
//
// 功能说明：
// - 使用 AES-256-GCM 加密算法保护敏感数据
// - 密钥管理：首次运行生成随机密钥并保存到本地文件
// - 支持字符串的加密和解密操作
//
// 安全特性：
// - AES-256-GCM：带认证的加密模式，防止篡改
// - 随机 nonce：每次加密使用不同的 nonce，确保相同明文产生不同密文
// - Base64 编码：密文以 base64 格式存储，便于数据库存储
//
// 密钥管理策略：
// - 密钥文件位置：~/.outlook-mail-manager/.key
// - 文件权限：0600（仅所有者可读写）
// - 密钥长度：32 字节（256 位）
// - 首次运行自动生成，后续运行从文件读取
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// EncryptionService 加密服务
//
// 提供数据加密和解密功能，使用 AES-256-GCM 算法
// 密钥在服务初始化时加载，整个应用生命周期内保持不变
type EncryptionService struct {
	key []byte // AES-256 密钥（32 字节）
}

// NewEncryptionService 创建加密服务实例
//
// 密钥管理流程：
// 1. 尝试从 ~/.outlook-mail-manager/.key 读取现有密钥
// 2. 如果文件不存在，生成新的 32 字节随机密钥
// 3. 将新密钥保存到文件（权限 0600）
// 4. 返回初始化完成的服务实例
//
// 返回值：
//   - *EncryptionService: 加密服务实例
//   - error: 密钥文件读写错误或密钥生成失败
//
// 注意事项：
//   - 密钥文件丢失会导致已加密数据无法解密
//   - 建议定期备份密钥文件
//   - 密钥文件应妥善保管，不要提交到版本控制系统
func NewEncryptionService() (*EncryptionService, error) {
	// 获取用户主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	// 构建密钥文件路径
	keyPath := filepath.Join(homeDir, ".outlook-mail-manager", ".key")

	// 尝试读取现有密钥
	if keyData, err := os.ReadFile(keyPath); err == nil {
		// 验证密钥长度
		if len(keyData) != 32 {
			return nil, errors.New("invalid key length: expected 32 bytes")
		}
		return &EncryptionService{key: keyData}, nil
	}

	// 密钥文件不存在，生成新密钥
	key := make([]byte, 32) // AES-256 需要 32 字节密钥
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	// 确保目录存在
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return nil, err
	}

	// 保存密钥到文件（权限 0600：仅所有者可读写）
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return nil, err
	}

	return &EncryptionService{key: key}, nil
}

// Encrypt 加密字符串
//
// 使用 AES-256-GCM 算法加密明文字符串
// 加密流程：
// 1. 创建 AES cipher block
// 2. 创建 GCM 模式（带认证的加密）
// 3. 生成随机 nonce（每次加密都不同）
// 4. 使用 GCM.Seal 加密数据（nonce + ciphertext + tag）
// 5. Base64 编码结果
//
// 参数：
//   - plaintext: 要加密的明文字符串
//
// 返回值：
//   - string: Base64 编码的密文（格式：base64(nonce + ciphertext + tag)）
//   - error: 加密过程中的错误
//
// 特殊处理：
//   - 空字符串直接返回空字符串（不加密）
//   - 相同明文每次加密产生不同密文（因为 nonce 随机）
func (s *EncryptionService) Encrypt(plaintext string) (string, error) {
	// 空字符串不需要加密
	if plaintext == "" {
		return "", nil
	}

	// 创建 AES cipher block
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	// 创建 GCM 模式（Galois/Counter Mode）
	// GCM 提供加密和认证，防止密文被篡改
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成随机 nonce（Number used ONCE）
	// nonce 长度由 GCM 决定（通常是 12 字节）
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 加密数据
	// Seal 将 nonce 作为前缀，然后是密文和认证标签
	// 格式：nonce || ciphertext || tag
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Base64 编码，便于存储到数据库
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密字符串
//
// 解密由 Encrypt 方法加密的密文
// 解密流程：
// 1. Base64 解码密文
// 2. 提取 nonce（前 NonceSize 字节）
// 3. 提取密文和认证标签（剩余字节）
// 4. 使用 GCM.Open 解密并验证
// 5. 返回明文字符串
//
// 参数：
//   - ciphertext: Base64 编码的密文
//
// 返回值：
//   - string: 解密后的明文字符串
//   - error: 解密失败或认证失败
//
// 特殊处理：
//   - 空字符串直接返回空字符串
//   - 密文被篡改会导致认证失败
//   - 使用错误的密钥会导致解密失败
func (s *EncryptionService) Decrypt(ciphertext string) (string, error) {
	// 空字符串不需要解密
	if ciphertext == "" {
		return "", nil
	}

	// Base64 解码
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// 创建 AES cipher block
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	// 创建 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 验证密文长度
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// 提取 nonce 和密文
	nonce := data[:nonceSize]
	ciphertextBytes := data[nonceSize:]

	// 解密并验证
	// Open 会验证认证标签，如果密文被篡改会返回错误
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
