// Package hash 提供密码哈希与校验。
package hash

import "golang.org/x/crypto/bcrypt"

// Cost bcrypt 计算强度。
const Cost = 10

// Make 生成 bcrypt 哈希。
func Make(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), Cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Check 校验明文与哈希是否匹配。
func Check(hashed, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}
