// GitCredService 私有仓库凭据库（M26 P2）：预配置 https 账密/token 或 ssh 私钥，按 host 自动匹配。
// Secret 不回传（model json:"-"）；列表/详情不泄露，任务克隆时才由 core 取用下发 agent。
package service

import (
	"context"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

var gitCredHostRe = regexp.MustCompile(`^(?:[a-zA-Z][a-zA-Z0-9+.-]*://)?(?:[^@/]*@)?([^/:?#]+)`)

type GitCredService struct {
	DB *gorm.DB
}

func NewGitCredService(db *gorm.DB) *GitCredService {
	return &GitCredService{DB: db}
}

func (s *GitCredService) List(ctx context.Context) ([]model.GitCredential, error) {
	out := make([]model.GitCredential, 0)
	if err := s.DB.WithContext(ctx).Order("host asc, id asc").Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (s *GitCredService) Create(ctx context.Context, name, typ, host, username, secret, remark string) (*model.GitCredential, error) {
	name, typ, host, username = strings.TrimSpace(name), strings.TrimSpace(typ), strings.ToLower(strings.TrimSpace(host)), strings.TrimSpace(username)
	if name == "" || len(name) > 64 {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "凭据名称不能为空且不超过 64 字")
	}
	if typ != "token" && typ != "ssh" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "凭据类型仅支持 token/ssh")
	}
	if host == "" || len(host) > 128 {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "匹配主机不能为空（可用 * 兜底）")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, errs.New(errs.CodeBadRequest, "error.badRequest", "凭据内容（密码/token/私钥）不能为空")
	}
	c := &model.GitCredential{Name: name, Type: typ, Host: host, Username: username, Secret: secret, Remark: strings.TrimSpace(remark)}
	if err := s.DB.WithContext(ctx).Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// Update 零值字段不更新（secret 传空表示保留原值）。
func (s *GitCredService) Update(ctx context.Context, id uint, name, host, username, secret, remark *string) error {
	c, err := s.byID(ctx, id)
	if err != nil {
		return err
	}
	if name != nil && strings.TrimSpace(*name) != "" {
		c.Name = strings.TrimSpace(*name)
	}
	if host != nil && strings.TrimSpace(*host) != "" {
		c.Host = strings.ToLower(strings.TrimSpace(*host))
	}
	if username != nil {
		c.Username = strings.TrimSpace(*username)
	}
	if remark != nil {
		c.Remark = strings.TrimSpace(*remark)
	}
	if secret != nil && strings.TrimSpace(*secret) != "" {
		c.Secret = *secret
	}
	return s.DB.WithContext(ctx).Save(c).Error
}

func (s *GitCredService) Delete(ctx context.Context, id uint) error {
	return s.DB.WithContext(ctx).Delete(&model.GitCredential{}, id).Error
}

// Match 按 git 地址的 host 自动匹配凭据：精确 host 优先，"*" 兜底；无命中返回 nil。
func (s *GitCredService) Match(ctx context.Context, gitURL string) (*model.GitCredential, error) {
	host := HostOfGitURL(gitURL)
	if host == "" {
		return nil, nil
	}
	var list []model.GitCredential
	if err := s.DB.WithContext(ctx).Where("host IN ?", []string{host, "*"}).Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	for _, c := range list {
		if c.Host == host {
			return &c, nil
		}
	}
	if len(list) > 0 {
		return &list[0], nil
	}
	return nil, nil
}

// GetByID 取单个凭据（含 secret，仅内部使用：任务克隆时解析下发）。
func (s *GitCredService) GetByID(ctx context.Context, id uint) (*model.GitCredential, error) {
	return s.byID(ctx, id)
}

func (s *GitCredService) byID(ctx context.Context, id uint) (*model.GitCredential, error) {
	var c model.GitCredential
	if err := s.DB.WithContext(ctx).First(&c, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "凭据不存在")
	}
	return &c, nil
}

// HostOfGitURL 提取 git 地址主机（https://u:p@github.com/x → github.com；git@github.com:a.git → github.com）。
func HostOfGitURL(gitURL string) string {
	m := gitCredHostRe.FindStringSubmatch(strings.TrimSpace(gitURL))
	if len(m) < 2 {
		return ""
	}
	return strings.ToLower(m[1])
}
