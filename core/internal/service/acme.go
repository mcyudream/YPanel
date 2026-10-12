// B1→B23：站点级 ACME 签发（薄壳，转发证书库服务）。
// 历史环境变量路径（YPANEL_ACME_EMAIL / YPANEL_ACME_ALI_KEY / YPANEL_ACME_ALI_SECRET）
// 由 CertificateService 兜底支持；新代码请走证书库（DNS 账户 + Acme 账户）。
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// AcmeService 站点级 ACME 签发入口（转发证书库）。
type AcmeService struct {
	sites *SiteService
	certs *CertificateService
}

// NewAcmeService 创建站点级 ACME 服务。
func NewAcmeService(sites *SiteService, certs *CertificateService) *AcmeService {
	return &AcmeService{sites: sites, certs: certs}
}

// IssueACMETask 为站点签发证书并启用 HTTPS（任务版：返回任务 ID，签发过程任务中心实时日志）。
// domain 留空用站点主域名，自动附带泛域名；DNS 账户优先：站点绑定域名匹配的证书库
// DNS 账户不存在时，回退面板环境变量。
func (s *AcmeService) IssueACMETask(ctx context.Context, siteID uint, domain string) (map[string]any, error) {
	if err := s.ensureService(); err != nil {
		return nil, err
	}
	site, err := s.sites.siteByID(siteID)
	if err != nil {
		return nil, err
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = site.Domain
	}
	// 环境变量路径兼容：未配置 DNS 账户时用 env 兜底（与 B1 行为一致）
	var dnsAccountID uint
	var rows []model.DnsAccount
	_ = s.certs.db.Limit(1).Find(&rows).Error
	if len(rows) > 0 {
		dnsAccountID = rows[0].ID
	}
	in := CertIssueInput{
		Domain:       domain,
		AutoRenew:    true,
		DnsAccountID: dnsAccountID,
	}
	// 预备段同步快速失败（域名/凭据/重名检查），任务体只承载耗时执行 + 站点 HTTPS 绑定
	p, err := s.certs.prepareIssue(ctx, in)
	if err != nil {
		return nil, err
	}
	if s.certs.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	if s.certs.tasks.HasRunning([]string{TaskCertIssue}, p.main) {
		return nil, errs.Wrap(errs.ErrBadRequest, "该域名已有签发任务运行中，请稍后再试")
	}
	task, err := s.certs.tasks.StartTask(TaskCertIssue,
		fmt.Sprintf("站点签发证书 %s（%s）", p.main, site.Name), p.main, 15*time.Minute,
		func(ctx context.Context, logf TaskLogf) error {
			cert, err := s.certs.issueExecute(ctx, logf, p)
			if err != nil {
				return err
			}
			logf("info", "绑定站点 HTTPS（site #%d %s）…", siteID, site.Name)
			return s.sites.ApplyCert(ctx, site, cert.ID, cert.CertName)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// ensureService 校验证书库可用（未装配时报错）。
func (s *AcmeService) ensureService() error {
	if s.certs == nil {
		return errs.New(errs.CodeInternal, "error.internal", "证书库服务未装配")
	}
	return nil
}
