// Package secretbox 提供 AES-256-GCM 字符串加解密，用于 SSH 凭证落库。
// 密钥来自环境变量 PAAS_AES_KEY，任意长度字符串，内部做 SHA-256 归一化。
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Encrypt 加密明文，输出 base64(nonce | ciphertext)。
func Encrypt(key, plaintext string) (string, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt 解密 Encrypt 的输出，密钥不符或数据损坏时返回错误。
func Decrypt(key, ciphertext string) (string, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}
	if len(raw) < aead.NonceSize() {
		return "", errors.New("密文长度不合法")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plain), nil
}

func newAEAD(key string) (cipher.AEAD, error) {
	if key == "" {
		return nil, errors.New("加密密钥未配置，请设置环境变量 PAAS_AES_KEY")
	}
	keyBytes := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return nil, fmt.Errorf("init aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}
	return aead, nil
}
