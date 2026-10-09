// 证书库服务（B23，对齐 1Panel 证书页）：证书全生命周期独立于站点管理。
// ACME 走 acme.sh + DNS 账户（aliyun/dnspod/cloudflare）；上传/自签直接落 certs 卷。
// 过期探测经 openssl x509，状态 ok/expiring/expired 在 List 时刷新。
package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// CertificateService 证书库服务。
type CertificateService struct {
	db     *gorm.DB
	nodes  *NodeService
	sites  *SiteService
	aesKey []byte
	envf   func(string) string // 环境变量读取（面板 systemd 注入路径，可选）
}

// NewCertificateService 创建证书库服务（凭据加密密钥由 JWT 密钥派生）。
func NewCertificateService(db *gorm.DB, nodes *NodeService, sites *SiteService, jwtSecret string) *CertificateService {
	sum := sha256.Sum256([]byte("ypanel-dnscred:" + jwtSecret))
	return &CertificateService{db: db, nodes: nodes, sites: sites, aesKey: sum[:]}
}

var certNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]{0,253}$`)

// ---- 通用执行 ----

func (s *CertificateService) run(ctx context.Context, cmd string, timeout int) (*dto.ExecResp, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	return agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeout})
}

// shQuote shell 单引号安全包裹。
func shQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'"'"'`) + "'"
}

// ---- DNS 账户凭据加解密 ----

func (s *CertificateService) encryptSecret(plain string) (string, error) {
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	enc := gcm.Seal(nil, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(append(nonce, enc...)), nil
}

func (s *CertificateService) decryptSecret(enc string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.aesKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errs.New(errs.CodeInternal, "error.internal", "凭据数据损坏")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// dnsProviderEnv DNS 账户 → acme.sh dns API 名称与环境变量前缀。
type dnsProviderEnv struct {
	dnsAPI string
	env    [][2]string // key=env 名，value=取值
}

func dnsProviderOf(p, accessKey, secret string) (dnsProviderEnv, error) {
	switch p {
	case "aliyun":
		return dnsProviderEnv{dnsAPI: "dns_ali", env: [][2]string{{"Ali_Key", accessKey}, {"Ali_Secret", secret}}}, nil
	case "dnspod":
		return dnsProviderEnv{dnsAPI: "dns_dp", env: [][2]string{{"DP_Id", accessKey}, {"DP_Key", secret}}}, nil
	case "cloudflare":
		return dnsProviderEnv{dnsAPI: "dns_cf", env: [][2]string{{"CF_Email", accessKey}, {"CF_Key", secret}}}, nil
	default:
		return dnsProviderEnv{}, errs.Wrap(errs.ErrBadRequest, "不支持的 DNS 服务商: "+p)
	}
}

// ---- acme.sh 基础（自 acme.go 迁移扩展）----

const acmeBin = "/root/.acme.sh/acme.sh"

func (s *CertificateService) ensureInstalled(ctx context.Context, email string) error {
	out, err := s.run(ctx, fmt.Sprintf("test -x %s && echo ok", acmeBin), 15)
	if err == nil && strings.TrimSpace(out.Output) == "ok" {
		return nil
	}
	install, ierr := s.run(ctx, fmt.Sprintf("curl -s https://get.acme.sh | sh -s email=%s", email), 180)
	if ierr != nil {
		return ierr
	}
	if install.ExitCode != 0 {
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "acme.sh 安装失败: "+firstLine(install.Output))
	}
	return nil
}

// caServer CA 类型 → acme.sh --server 值。
func caServer(ca string) string {
	switch ca {
	case "zerossl":
		return "zerossl"
	case "buypass":
		return "buypass"
	default:
		return "letsencrypt"
	}
}

// ---- 证书库 CRUD ----

// CertItem 列表视图。
type CertItem struct {
	model.Certificate
	Sites []string `json:"sites"` // 绑定的站点名
}

// List 证书列表（刷新探测与状态）。
func (s *CertificateService) List(ctx context.Context) ([]CertItem, error) {
	var rows []model.Certificate
	if err := s.db.Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	// 站点绑定反查
	var sites []model.Site
	_ = s.db.Select("name, cert_id, cert_domain").Find(&sites).Error
	bind := map[uint][]string{}
	for _, st := range sites {
		if st.CertID != 0 {
			bind[st.CertID] = append(bind[st.CertID], st.Name)
		}
	}
	out := make([]CertItem, 0, len(rows))
	var changed []model.Certificate // 仅探测结果有变化的行回写（Save 空切片报 ErrEmptySlice，也避免每次开列表页全表写）
	for i := range rows {
		r := &rows[i]
		before := *r
		s.probe(ctx, r)
		refreshCertStatus(r)
		if certProbeChanged(&before, r) {
			changed = append(changed, *r)
		}
		out = append(out, CertItem{Certificate: *r, Sites: bind[r.ID]})
	}
	if len(changed) > 0 {
		if err := s.db.Save(changed).Error; err != nil {
			return nil, err
		}
	}
	return out, nil
}

// probe 探测证书文件补齐 NotAfter/Issuer（无文件则 status=error）。
func (s *CertificateService) probe(ctx context.Context, c *model.Certificate) {
	crt := path.Join(nginxCertDir, c.CertName+".crt")
	out, err := s.run(ctx, "openssl x509 -in "+shQuote(crt)+" -noout -enddate -issuer", 20)
	if err != nil || out.ExitCode != 0 {
		if c.NotAfter == nil {
			c.Status = "error"
		}
		return
	}
	var notAfter *time.Time
	issuer := ""
	for _, line := range strings.Split(out.Output, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "notAfter="); ok {
			if t, e := time.Parse("Jan _2 15:04:05 2006 MST", strings.TrimSpace(v)); e == nil {
				notAfter = &t
			}
		} else if v, ok := strings.CutPrefix(line, "issuer="); ok {
			issuer = parseIssuerOrg(v)
		}
	}
	if notAfter != nil {
		c.NotAfter = notAfter
	}
	if issuer != "" {
		c.Issuer = issuer
	}
}

// parseIssuerOrg 从 openssl issuer 串提取组织（优先 O=，回退 CN=）。
func parseIssuerOrg(raw string) string {
	for _, key := range []string{"O=", "CN="} {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if v, ok := strings.CutPrefix(part, key); ok {
				return strings.Trim(v, "'\"")
			}
		}
	}
	return ""
}

// refreshStatus 按剩余有效期刷新状态。
func refreshCertStatus(c *model.Certificate) {
	if c.Status == "error" {
		return
	}
	if c.NotAfter == nil {
		c.Status = "ok"
		return
	}
	days := time.Until(*c.NotAfter)
	switch {
	case days < 0:
		c.Status = "expired"
	case days < 15*24*time.Hour:
		c.Status = "expiring"
	default:
		c.Status = "ok"
	}
}

// certProbeChanged 判断探测/状态刷新是否改动了需落库的字段（NotAfter 为指针，按值比较）。
func certProbeChanged(before, after *model.Certificate) bool {
	if before.Status != after.Status || before.Issuer != after.Issuer {
		return true
	}
	if (before.NotAfter == nil) != (after.NotAfter == nil) {
		return true
	}
	return after.NotAfter != nil && !after.NotAfter.Equal(*before.NotAfter)
}

// Detail 证书详情（openssl -text 全文）。
func (s *CertificateService) Detail(ctx context.Context, id uint) (map[string]any, error) {
	c, err := s.byID(id)
	if err != nil {
		return nil, err
	}
	crt := path.Join(nginxCertDir, c.CertName+".crt")
	out, err := s.run(ctx, "openssl x509 -in "+shQuote(crt)+" -noout -text", 20)
	text := ""
	if err == nil && out.ExitCode == 0 {
		text = out.Output
	}
	alt := []string{}
	if c.AltDomains != "" {
		_ = json.Unmarshal([]byte(c.AltDomains), &alt)
	}
	return map[string]any{"cert": c, "altDomains": alt, "text": text}, nil
}

// CertUpdateInput 编辑输入（备注/自动续签）。
type CertUpdateInput struct {
	Remark    *string `json:"remark"`
	AutoRenew *bool   `json:"autoRenew"`
}

// Update 编辑证书（备注/自动续签；关闭自动续签将证书移出 acme.sh 续期列表）。
func (s *CertificateService) Update(ctx context.Context, id uint, in CertUpdateInput) error {
	c, err := s.byID(id)
	if err != nil {
		return err
	}
	if in.Remark != nil {
		c.Remark = strings.TrimSpace(*in.Remark)
	}
	if in.AutoRenew != nil {
		c.AutoRenew = *in.AutoRenew
	}
	if c.Provider == "acme" {
		if c.AutoRenew {
			if err := s.reinstallCertHook(ctx, c); err != nil {
				return err
			}
		} else {
			if out, err := s.run(ctx, fmt.Sprintf("%s --remove -d %s --ecc", acmeBin, c.Domain), 60); err == nil && out.ExitCode != 0 {
				// 移除失败不阻断（可能本就未挂载）
				_ = out
			}
		}
	}
	return s.db.Save(c).Error
}

// Delete 删除证书（站点绑定时拒绝）。
func (s *CertificateService) Delete(ctx context.Context, id uint) error {
	c, err := s.byID(id)
	if err != nil {
		return err
	}
	var n int64
	_ = s.db.Model(&model.Site{}).Where("cert_id = ?", id).Count(&n).Error
	if n > 0 {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("证书仍被 %d 个站点绑定，请先在站点 HTTPS 设置中解绑", n))
	}
	if c.Provider == "acme" {
		_, _ = s.run(ctx, fmt.Sprintf("%s --remove -d %s --ecc", acmeBin, c.Domain), 60)
	}
	_, _ = s.run(ctx, fmt.Sprintf("rm -f %s %s",
		shQuote(path.Join(nginxCertDir, c.CertName+".crt")),
		shQuote(path.Join(nginxCertDir, c.CertName+".key"))), 20)
	return s.db.Delete(c).Error
}

// Renew 手动续签（ACME 证书）。
func (s *CertificateService) Renew(ctx context.Context, id uint) error {
	c, err := s.byID(id)
	if err != nil {
		return err
	}
	if c.Provider != "acme" {
		return errs.Wrap(errs.ErrBadRequest, "仅 ACME 证书支持续签（上传/自签证书请重新上传或签发）")
	}
	creds, err := s.dnsCredsFor(c)
	if err != nil {
		return err
	}
	out, err := s.run(ctx, creds+fmt.Sprintf("%s --renew -d %s --ecc --force", acmeBin, c.Domain), 600)
	if err != nil {
		return err
	}
	c.IssueLog = out.Output
	if out.ExitCode != 0 {
		s.db.Save(c)
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "续签失败: "+firstLine(out.Output))
	}
	if err := s.reinstallCertHook(ctx, c); err != nil {
		return err
	}
	s.probe(ctx, c)
	refreshCertStatus(c)
	return s.db.Save(c).Error
}

// dnsCredsFor 生成证书续签所需的 DNS 凭据环境前缀。
func (s *CertificateService) dnsCredsFor(c *model.Certificate) (string, error) {
	if c.DnsAccountID == 0 {
		return "", nil // 凭据来自环境变量（systemd 注入），续签时 acme.sh 自行读取
	}
	var acc model.DnsAccount
	if err := s.db.First(&acc, c.DnsAccountID).Error; err != nil {
		return "", errs.Wrap(errs.ErrBadRequest, "DNS 账户不存在")
	}
	secret, err := s.decryptSecret(acc.SecretEnc)
	if err != nil {
		return "", err
	}
	p, err := dnsProviderOf(acc.Provider, acc.AccessKey, secret)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, kv := range p.env {
		fmt.Fprintf(&b, "%s=%s ", kv[0], shQuote(kv[1]))
	}
	return b.String(), nil
}

// reinstallCertHook 重新挂载 install-cert 钩子（续期自动 reload nginx）。
func (s *CertificateService) reinstallCertHook(ctx context.Context, c *model.Certificate) error {
	keyPath := path.Join(nginxCertDir, c.CertName+".key")
	crtPath := path.Join(nginxCertDir, c.CertName+".crt")
	out, err := s.run(ctx, fmt.Sprintf(
		"%s --install-cert -d %s --ecc --key-file %s --fullchain-file %s --reloadcmd 'docker exec ypanel-nginx nginx -s reload'",
		acmeBin, c.Domain, keyPath, crtPath), 120)
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "证书部署钩子安装失败: "+firstLine(out.Output))
	}
	return nil
}

func (s *CertificateService) byID(id uint) (*model.Certificate, error) {
	var c model.Certificate
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.notFound", "证书不存在")
	}
	return &c, nil
}

// normalizeDomains 清洗域名列表（支持泛域名 *.x）。
func normalizeDomains(domains []string) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(d, "*.")))
		if d == "" {
			continue
		}
		if !domainPattern.MatchString(d) {
			return nil, errs.Wrap(errs.ErrBadRequest, "域名不合法: "+d)
		}
		if seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out, nil
}

// saveAltDomains 其他域名 JSON 序列化。
func saveAltDomains(domains []string) string {
	if len(domains) == 0 {
		return ""
	}
	b, _ := json.Marshal(domains)
	return string(b)
}

// loadAltDomains 反序列化其他域名。
func loadAltDomains(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
