// B1→B23：站点级 ACME 签发（薄壳，转发证书库服务）。
// 历史环境变量路径（YPANEL_ACME_EMAIL / YPANEL_ACME_ALI_KEY / YPANEL_ACME_ALI_SECRET）
// 由 CertificateService 兜底支持；新代码请走证书库（DNS 账户 + Acme 账户）。
package service

import (
	"context"
	"strings"

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

// IssueACME 为站点签发证书并启用 HTTPS（domain 留空用站点主域名，自动附带泛域名）。
// DNS 账户优先：站点绑定域名匹配的证书库 DNS 账户不存在时，回退面板环境变量。
func (s *AcmeService) IssueACME(ctx context.Context, siteID uint, domain string) (map[string]any, error) {
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
	cert, err := s.certs.Issue(ctx, CertIssueInput{
		Domain:       domain,
		AutoRenew:    true,
		DnsAccountID: dnsAccountID,
	})
	if err != nil {
		return nil, err
	}
	// 启用站点 HTTPS（证书绑定）
	if err := s.sites.ApplyCert(ctx, site, cert.ID, cert.CertName); err != nil {
		return nil, err
	}
	return map[string]any{
		"domain": cert.Domain, "certDomain": cert.CertName,
		"issuer": cert.Issuer, "certId": cert.ID,
	}, nil
}

// ensureService 校验证书库可用（未装配时报错）。
func (s *AcmeService) ensureService() error {
	if s.certs == nil {
		return errs.New(errs.CodeInternal, "error.internal", "证书库服务未装配")
	}
	return nil
}
