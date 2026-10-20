package service

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/hash"
	"github.com/Mike09811/Toze/server/internal/pkg/token"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// AuthService 认证服务。
type AuthService struct {
	users  *repository.UserRepository
	roles  *repository.RoleRepository
	tokens *repository.TokenRepository
	jwt    *token.Manager
	log    *slog.Logger
}

// NewAuthService 构造函数。
func NewAuthService(users *repository.UserRepository, roles *repository.RoleRepository,
	tokens *repository.TokenRepository, jwt *token.Manager, log *slog.Logger) *AuthService {
	return &AuthService{users: users, roles: roles, tokens: tokens, jwt: jwt, log: log}
}

// Login 账号密码登录。
func (s *AuthService) Login(account, password string) (*LoginResult, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		return nil, errcode.New(errcode.InvalidParams, "请输入账号和密码")
	}

	user, err := s.users.FindByAccount(account)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.LoginFailed, "")
		}
		return nil, err
	}
	if !hash.Check(user.Password, password) {
		return nil, errcode.New(errcode.LoginFailed, "")
	}
	if user.Status != 1 {
		return nil, errcode.New(errcode.UserDisabled, "")
	}
	return s.issue(user)
}

// Register 注册普通用户并直接登录。
func (s *AuthService) Register(account, password, name string) (*LoginResult, error) {
	account = strings.TrimSpace(account)
	if len(account) < 3 {
		return nil, errcode.New(errcode.InvalidParams, "账号至少 3 位")
	}
	if len(password) < 6 {
		return nil, errcode.New(errcode.InvalidParams, "密码至少 6 位")
	}
	exists, err := s.users.ExistsByAccount(account, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errcode.New(errcode.UserExists, "")
	}

	hashed, err := hash.Make(password)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = account
	}
	user := &model.User{
		UUID:     NewUUID(),
		Account:  account,
		Password: hashed,
		Name:     name,
		Status:   1,
		Gender:   1,
		Region:   []string{},
	}

	roles, err := s.roles.FindByNames([]string{"ROLE_USER"})
	if err != nil {
		return nil, err
	}
	if err := s.users.Create(user, roles); err != nil {
		return nil, err
	}
	created, err := s.users.FindByID(user.ID)
	if err != nil {
		return nil, err
	}
	return s.issue(created)
}

// Logout 注销令牌（加入黑名单）。
func (s *AuthService) Logout(jti string, exp time.Time) error {
	if jti == "" {
		return nil
	}
	if exp.IsZero() {
		exp = time.Now().Add(s.jwt.TTL())
	}
	return s.tokens.Revoke(jti, exp)
}

// Profile 查询当前用户信息。
func (s *AuthService) Profile(userID uint64) (*ProfileDTO, error) {
	user, err := s.users.FindByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "用户不存在")
		}
		return nil, err
	}
	return &ProfileDTO{
		UUID:    user.UUID,
		Name:    user.Name,
		Avatar:  user.Avatar,
		Access:  user.RoleCodes(),
		Account: user.Account,
		Phone:   user.Phone,
		Mail:    user.Mail,
		Gender:  user.Gender,
		Desc:    user.Desc,
	}, nil
}

// ChangePassword 修改密码。
func (s *AuthService) ChangePassword(userID uint64, oldPwd, newPwd string) error {
	if len(newPwd) < 6 {
		return errcode.New(errcode.InvalidParams, "新密码至少 6 位")
	}
	user, err := s.users.FindByID(userID)
	if err != nil {
		return err
	}
	if !hash.Check(user.Password, oldPwd) {
		return errcode.New(errcode.OldPasswordErr, "")
	}
	hashed, err := hash.Make(newPwd)
	if err != nil {
		return err
	}
	return s.users.UpdatePassword(userID, hashed)
}

func (s *AuthService) issue(user *model.User) (*LoginResult, error) {
	roles := user.RoleCodes()
	tk, exp, err := s.jwt.Generate(user.ID, user.UUID, roles)
	if err != nil {
		return nil, err
	}
	return &LoginResult{
		Username: user.Account,
		UUID:     user.UUID,
		Info: ProfileDTO{
			UUID:    user.UUID,
			Name:    user.Name,
			Avatar:  user.Avatar,
			Access:  roles,
			Account: user.Account,
			Phone:   user.Phone,
			Mail:    user.Mail,
			Gender:  user.Gender,
			Desc:    user.Desc,
		},
		Token:     tk,
		ExpiresIn: int64(time.Until(exp).Seconds()),
	}, nil
}

// HasRole 判断用户是否具备任一角色。
func HasRole(roles []string, required ...string) bool {
	for _, r := range roles {
		for _, req := range required {
			if r == req {
				return true
			}
		}
	}
	return false
}

// ParseID 字符串 ID 转 uint64。
func ParseID(s string) (uint64, error) {
	if s == "" {
		return 0, errcode.New(errcode.InvalidParams, "ID 不能为空")
	}
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, errcode.New(errcode.InvalidParams, "ID 格式错误")
	}
	return id, nil
}
