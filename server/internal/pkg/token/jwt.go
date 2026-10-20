// Package token 负责 JWT 的签发与校验。
package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalid 令牌非法。
	ErrInvalid = errors.New("invalid token")
	// ErrExpired 令牌过期。
	ErrExpired = errors.New("token expired")
)

// Claims 自定义载荷。
type Claims struct {
	UserID uint64   `json:"uid"`
	UUID   string   `json:"uuid"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

// Manager 签发与解析令牌。
type Manager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// New 创建令牌管理器。
func New(secret, issuer string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), issuer: issuer, ttl: ttl}
}

// TTL 返回令牌有效期。
func (m *Manager) TTL() time.Duration { return m.ttl }

// Generate 签发令牌，返回令牌与过期时间。
func (m *Manager) Generate(userID uint64, uuid string, roles []string) (string, time.Time, error) {
	now := time.Now()
	expire := now.Add(m.ttl)
	claims := Claims{
		UserID: userID,
		UUID:   uuid,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   uuid,
			ID:        newJTI(),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expire),
		},
	}
	tk := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tk.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expire, nil
}

// Parse 校验并解析令牌。
func (m *Manager) Parse(raw string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalid
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired())

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, ErrInvalid
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalid
	}
	return claims, nil
}

func newJTI() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b)
}
