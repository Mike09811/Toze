package service

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/Mike09811/Toze/server/internal/model"
	"github.com/Mike09811/Toze/server/internal/pkg/errcode"
	"github.com/Mike09811/Toze/server/internal/pkg/hash"
	"github.com/Mike09811/Toze/server/internal/pkg/pagination"
	"github.com/Mike09811/Toze/server/internal/repository"
)

// UserListQuery 用户列表查询条件。
type UserListQuery struct {
	Name     string
	Phone    string
	Mail     string
	Account  string
	ID       string
	Gender   int
	Status   int
	Date     []string
	Offset   int
	Limit    int
	SortType string
	SortKey  string
}

// UserUpsert 新增 / 编辑用户入参。
type UserUpsert struct {
	ID       string
	Name     string
	Account  string
	Password string
	Phone    string
	Mail     string
	Avatar   string
	Gender   int
	Status   int
	Birthday string
	Region   []string
	Role     []string
	Desc     string
}

// UserService 用户业务逻辑。
type UserService struct {
	users *repository.UserRepository
	roles *repository.RoleRepository
	log   *slog.Logger
}

// NewUserService 构造函数。
func NewUserService(users *repository.UserRepository, roles *repository.RoleRepository, log *slog.Logger) *UserService {
	return &UserService{users: users, roles: roles, log: log}
}

// List 分页查询。
func (s *UserService) List(q UserListQuery) ([]UserDTO, int64, error) {
	filter := repository.UserFilter{
		Name: q.Name, Phone: q.Phone, Mail: q.Mail, Account: q.Account,
		ID: q.ID, Gender: q.Gender, Status: q.Status,
	}
	if len(q.Date) == 2 {
		if from, ok := parseDate(q.Date[0], false); ok {
			filter.DateFrom = &from
		}
		if to, ok := parseDate(q.Date[1], true); ok {
			filter.DateTo = &to
		}
	}
	p := pagination.Parse(q.Offset, q.Limit, q.SortKey, q.SortType)
	users, total, err := s.users.List(filter, p)
	if err != nil {
		return nil, 0, err
	}
	return ToUserDTOs(users), total, nil
}

// Create 新增用户。
func (s *UserService) Create(req UserUpsert) (*UserDTO, error) {
	name := strings.TrimSpace(req.Name)
	account := strings.TrimSpace(req.Account)
	if name == "" {
		return nil, errcode.New(errcode.InvalidParams, "用户名不能为空")
	}
	if len(account) < 3 {
		return nil, errcode.New(errcode.InvalidParams, "登录账号至少 3 位")
	}
	exists, err := s.users.ExistsByAccount(account, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errcode.New(errcode.UserExists, "")
	}

	pwd := req.Password
	if pwd == "" {
		pwd = "123456"
	}
	hashed, err := hash.Make(pwd)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		UUID:     NewUUID(),
		Account:  account,
		Password: hashed,
		Name:     name,
		Avatar:   req.Avatar,
		Phone:    req.Phone,
		Mail:     req.Mail,
		Gender:   defaultInt(req.Gender, 1),
		Status:   defaultInt(req.Status, 1),
		Birthday: req.Birthday,
		Region:   req.Region,
		Desc:     req.Desc,
	}
	if user.Region == nil {
		user.Region = []string{}
	}

	roles, err := s.resolveRoles(req.Role, true)
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
	dto := ToUserDTO(created)
	return &dto, nil
}

// Update 编辑用户。
func (s *UserService) Update(req UserUpsert) (*UserDTO, error) {
	id, err := ParseID(req.ID)
	if err != nil {
		return nil, err
	}
	user, err := s.users.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errcode.New(errcode.NotFound, "用户不存在")
		}
		return nil, err
	}

	account := strings.TrimSpace(req.Account)
	if account != "" && account != user.Account {
		exists, err := s.users.ExistsByAccount(account, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errcode.New(errcode.UserExists, "")
		}
		user.Account = account
	}
	if strings.TrimSpace(req.Name) != "" {
		user.Name = strings.TrimSpace(req.Name)
	}
	user.Phone = req.Phone
	user.Mail = req.Mail
	user.Avatar = req.Avatar
	user.Birthday = req.Birthday
	user.Desc = req.Desc
	if req.Gender > 0 {
		user.Gender = req.Gender
	}
	if req.Status > 0 {
		user.Status = req.Status
	}
	if req.Region != nil {
		user.Region = req.Region
	}
	if req.Password != "" {
		if len(req.Password) < 6 {
			return nil, errcode.New(errcode.InvalidParams, "密码至少 6 位")
		}
		hashed, err := hash.Make(req.Password)
		if err != nil {
			return nil, err
		}
		user.Password = hashed
	}
	roles, err := s.resolveRoles(req.Role, false)
	if err != nil {
		return nil, err
	}
	if err := s.users.Update(user, roles); err != nil {
		return nil, err
	}
	updated, err := s.users.FindByID(id)
	if err != nil {
		return nil, err
	}
	dto := ToUserDTO(updated)
	return &dto, nil
}

// Delete 删除用户。
func (s *UserService) Delete(id, operatorID uint64) error {
	if id == operatorID {
		return errcode.New(errcode.Forbidden, "不能删除当前登录账号")
	}
	user, err := s.users.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return errcode.New(errcode.NotFound, "用户不存在")
		}
		return err
	}
	if user.Account == "admin" {
		return errcode.New(errcode.Forbidden, "内置管理员不可删除")
	}
	return s.users.Delete(id)
}

// DeleteMany 批量删除。
func (s *UserService) DeleteMany(ids []uint64, operatorID uint64) error {
	for _, id := range ids {
		if id == operatorID {
			return errcode.New(errcode.Forbidden, "不能删除当前登录账号")
		}
		u, err := s.users.FindByID(id)
		if err != nil {
			continue
		}
		if u.Account == "admin" {
			return errcode.New(errcode.Forbidden, "内置管理员不可删除")
		}
	}
	return s.users.DeleteMany(ids)
}

func (s *UserService) resolveRoles(names []string, useDefault bool) ([]model.Role, error) {
	if len(names) == 0 {
		if !useDefault {
			return nil, nil
		}
		names = []string{"ROLE_USER"}
	}
	roles, err := s.roles.FindByNames(names)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, errcode.New(errcode.InvalidParams, "角色不存在")
	}
	return roles, nil
}

func defaultInt(v, def int) int {
	if v == 0 {
		return def
	}
	return v
}

func parseDate(s string, endOfDay bool) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02"}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, time.Local); err == nil {
			if endOfDay && l == "2006-01-02" {
				t = t.Add(24*time.Hour - time.Second)
			}
			return t, true
		}
	}
	return time.Time{}, false
}
