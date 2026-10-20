package service

import "github.com/Mike09811/Toze/server/internal/repository"

// I18nService 多语言服务。
type I18nService struct {
	repo *repository.I18nRepository
}

// NewI18nService 构造函数。
func NewI18nService(repo *repository.I18nRepository) *I18nService {
	return &I18nService{repo: repo}
}

// Messages 返回 locale -> module -> key -> value 的嵌套结构。
func (s *I18nService) Messages() (map[string]map[string]map[string]string, error) {
	items, err := s.repo.All()
	if err != nil {
		return nil, err
	}
	out := make(map[string]map[string]map[string]string)
	for _, item := range items {
		locale, ok := out[item.Locale]
		if !ok {
			locale = make(map[string]map[string]string)
			out[item.Locale] = locale
		}
		mod, ok := locale[item.Module]
		if !ok {
			mod = make(map[string]string)
			locale[item.Module] = mod
		}
		mod[item.Key] = item.Value
	}
	return out, nil
}
