// B1：ACME 证书（acme.sh 通道 + 阿里云 DNS API 挑战）。
// 流程：安装 acme.sh（首次）→ export Ali 凭据 → dns_ali 签发泛域名 → 证书安装到
// nginx certs 目录 → 站点启用 HTTPS（复用 CertDomain 链路）。自动续期由 acme.sh
// 内置 cron 完成，安装时已加 --install-cert reload 钩子。
package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// AcmeService ACME 证书签发。
type AcmeService struct {
	sites  *SiteService
	nodes  *NodeService
	getEnv func(string) string
}

// NewAcmeService 创建（getEnv 读取面板进程环境，凭据经 systemd env 注入，不入库）。
func NewAcmeService(sites *SiteService, nodes *NodeService, getEnv func(string) string) *AcmeService {
	return &AcmeService{sites: sites, nodes: nodes, getEnv: getEnv}
}

// acmeBin acme.sh 客户端路径。
const acmeBin = "/root/.acme.sh/acme.sh"

// run 组装并经 agent 执行 shell。
func (s *AcmeService) run(ctx context.Context, cmd string, timeout int) (*dto.ExecResp, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	return agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
}

// ensureInstalled 确认 acme.sh 存在，缺失则安装。
func (s *AcmeService) ensureInstalled(ctx context.Context) error {
	out, err := s.run(ctx, fmt.Sprintf("test -x %s && echo ok", acmeBin), 15)
	if err == nil && strings.TrimSpace(out.Output) == "ok" {
		return nil
	}
	install, ierr := s.run(ctx,
		fmt.Sprintf("curl -s https://get.acme.sh | sh -s email=%s", s.adminEmail()), 180)
	if ierr != nil {
		return ierr
	}
	if install.ExitCode != 0 {
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "acme.sh 安装失败: "+firstLine(install.Output))
	}
	return nil
}

func (s *AcmeService) adminEmail() string {
	if e := s.getEnv("YPANEL_ACME_EMAIL"); e != "" {
		return e
	}
	return "admin@yudream.cn"
}

// aliCreds 阿里云 DNS API 凭据（env 注入）。
func (s *AcmeService) aliCreds() (key, secret string, ok bool) {
	key = s.getEnv("YPANEL_ACME_ALI_KEY")
	secret = s.getEnv("YPANEL_ACME_ALI_SECRET")
	return key, secret, key != "" && secret != ""
}

// IssueACME 用 DNS API 挑战签发证书并部署到指定站点。
// domain 为主域名（如 yudream.cn），自动附带泛域名 *.domain。
func (s *AcmeService) IssueACME(ctx context.Context, siteID uint, domain string) (map[string]any, error) {
	site, err := s.sites.siteByID(siteID)
	if err != nil {
		return nil, err
	}
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = site.Domain
	}
	if domainPattern.MatchString(domain) == false && strings.HasPrefix(domain, "*.") == false {
		// 允许 *.example.com
		base := strings.TrimPrefix(domain, "*.")
		if domainPattern.MatchString(base) == false {
			return nil, errs.Wrap(errs.ErrBadRequest, "证书域名不合法: "+domain)
		}
	}
	aliKey, aliSecret, credOK := s.aliCreds()
	if !credOK {
		return nil, errs.Wrap(errs.ErrBadRequest, "未配置阿里云 DNS 凭据（YPANEL_ACME_ALI_KEY/YPANEL_ACME_ALI_SECRET）")
	}
	if err := s.ensureInstalled(ctx); err != nil {
		return nil, err
	}

	// 签发（DNS API 挑战，主域名 + 泛域名）
	issue := fmt.Sprintf(
		"Ali_Key='%s' Ali_Secret='%s' %s --issue --dns dns_ali -d %s -d '*.%s' --server letsencrypt --keylength ec-256",
		aliKey, aliSecret, acmeBin, domain, domain)
	out, err := s.run(ctx, issue, 600)
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "ACME 签发失败: "+firstLine(out.Output))
	}

	// 部署：安装到 nginx certs（certDomain=主域名），并加 reload 钩子（acme.sh 续期后自动重载）
	certName := strings.TrimPrefix(domain, "*.")
	keyPath := path.Join(nginxCertDir, certName+".key")
	crtPath := path.Join(nginxCertDir, certName+".crt")
	installCmd := fmt.Sprintf(
		"%s --install-cert -d %s --ecc --key-file %s --fullchain-file %s --reloadcmd 'docker exec ypanel-nginx nginx -s reload'",
		acmeBin, domain, keyPath, crtPath)
	inst, err := s.run(ctx, installCmd, 120)
	if err != nil {
		return nil, err
	}
	if inst.ExitCode != 0 {
		return nil, errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "证书安装失败: "+firstLine(inst.Output))
	}

	// 启用站点 HTTPS（CertDomain=证书文件名基准）
	site.CertDomain = certName
	if err := s.sitesUpdateCert(site, certName); err != nil {
		return nil, err
	}
	return map[string]any{
		"domain": domain, "certDomain": certName,
		"issuer": "Let's Encrypt", "at": time.Now().Format(time.RFC3339),
	}, nil
}

// sitesUpdateCert 更新站点证书域并重写配置。
func (s *AcmeService) sitesUpdateCert(site *model.Site, certDomain string) error {
	if err := s.sites.db.Model(site).Update("cert_domain", certDomain).Error; err != nil {
		return err
	}
	return s.sites.writeConf(context.Background(), site, confTemplate(site, true, parseWaf(site)))
}

var _ = time.Now
