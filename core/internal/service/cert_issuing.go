// 证书库：签发（ACME/DNS 挑战）、上传、自签与 DNS/Acme 账户管理。
package service

import (
	"context"

	"github.com/ypanel/core/internal/agentclient"
	"fmt"
	"path"
	"strings"
	"time"

	
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// CertIssueInput ACME 签发输入。
type CertIssueInput struct {
	Domain        string `json:"domain"`        // 主域名（自动附带 *.<domain> 泛域名）
	AltDomains    string `json:"altDomains"`    // 其他域名（换行/逗号分隔，可含 *.）
	Remark        string `json:"remark"`
	AutoRenew     bool   `json:"autoRenew"`
	AcmeAccountID uint   `json:"acmeAccountId"` // 空则用面板环境变量邮箱
	DnsAccountID  uint   `json:"dnsAccountId"`  // 空则尝试面板环境变量（阿里云）
	DeploySiteIDs []uint `json:"deploySiteIds"` // 签发成功后一键部署（绑定 HTTPS）到这些站点；空=仅入库
}

// issuePlan 签发预备产物：校验/凭据/命令装配完成后供 execute 阶段使用。
type issuePlan struct {
	in        CertIssueInput
	main      string
	alts      []string
	email     string
	credEnv   string
	dnsAPI    string
	ca        string
	keylen    string
	issueCmd  string
	certName  string
}

// prepareIssue 签发预备段：入参校验、账户/DNS 凭据装配、重名检查、组 acme.sh 命令。
// 全部为快速失败检查（同步段执行，域名/账户错误即时报给调用方，不进任务）。
func (s *CertificateService) prepareIssue(ctx context.Context, in CertIssueInput) (*issuePlan, error) {
	main := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(in.Domain, "*.")))
	if !domainPattern.MatchString(main) {
		return nil, errs.Wrap(errs.ErrBadRequest, "主域名不合法: "+in.Domain)
	}
	alts, err := normalizeDomains(strings.FieldsFunc(in.AltDomains, func(r rune) bool { return r == '\n' || r == ',' || r == ';' }))
	if err != nil {
		return nil, err
	}
	alts = append(alts, "*."+main) // 泛域名兜底

	// Acme 账户
	email := s.envEmail()
	ca := "letsencrypt"
	keylen := "ec-256"
	if in.AcmeAccountID != 0 {
		var acc model.AcmeAccount
		if err := s.db.First(&acc, in.AcmeAccountID).Error; err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "ACME 账户不存在")
		}
		email = acc.Email
		ca = caServer(acc.CAType)
		keylen = acc.KeyType
	}

	// DNS 凭据（账户优先，环境变量兜底）
	credEnv := ""
	dnsAPI := ""
	if in.DnsAccountID != 0 {
		var acc model.DnsAccount
		if err := s.db.First(&acc, in.DnsAccountID).Error; err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "DNS 账户不存在")
		}
		secret, e := s.decryptSecret(acc.SecretEnc)
		if e != nil {
			return nil, e
		}
		p, e := dnsProviderOf(acc.Provider, acc.AccessKey, secret)
		if e != nil {
			return nil, e
		}
		for _, kv := range p.env {
			credEnv += fmt.Sprintf("%s=%s ", kv[0], shQuote(kv[1]))
		}
		dnsAPI = p.dnsAPI
	} else {
		k, sec, ok := s.envAliCreds()
		if !ok {
			return nil, errs.Wrap(errs.ErrBadRequest, "请先在「DNS 账户」中配置 DNS 服务商 API，或在面板环境变量中配置 YPANEL_ACME_ALI_KEY/YPANEL_ACME_ALI_SECRET")
		}
		credEnv = fmt.Sprintf("Ali_Key=%s Ali_Secret=%s ", shQuote(k), shQuote(sec))
		dnsAPI = "dns_ali"
	}

	certName := main
	if _, err := s.byName(certName); err == nil {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "证书已存在（同主域名），请先删除旧证书")
	}

	// 签发命令：主域名 + 其他域名 + 泛域名
	var dArgs strings.Builder
	fmt.Fprintf(&dArgs, " -d %s", main)
	for _, d := range alts {
		fmt.Fprintf(&dArgs, " -d %s", d)
	}
	issue := fmt.Sprintf(acmeEnv+"%s%s --issue --dns %s%s --server %s --keylength %s",
		credEnv, acmeBin, dnsAPI, dArgs.String(), ca, keylen)
	return &issuePlan{in: in, main: main, alts: alts, email: email, credEnv: credEnv,
		dnsAPI: dnsAPI, ca: ca, keylen: keylen, issueCmd: issue, certName: certName}, nil
}

// issueExecute 签发执行段：acme.sh 安装检查 → DNS 挑战签发 → 入库 → 钩子/探测。
// 耗时主体（安装 180s + 签发 600s），logf 非 nil 时全程流式任务日志。
func (s *CertificateService) issueExecute(ctx context.Context, logf TaskLogf, p *issuePlan) (*model.Certificate, error) {
	if err := s.ensureInstalled(ctx, logf, p.email); err != nil {
		return nil, err
	}
	if logf != nil {
		logf("info", "开始签发（DNS 验证，最长 10 分钟）：主域名 %s，共 %d 个域名", p.main, len(p.alts))
	}
	out, code, err := s.runStream(ctx, p.issueCmd, 600, logf)
	if err != nil {
		return nil, err
	}
	c := &model.Certificate{
		CertName: p.certName, Domain: p.main,
		Provider: "acme", Remark: strings.TrimSpace(p.in.Remark), AutoRenew: p.in.AutoRenew,
		AcmeAccountID: p.in.AcmeAccountID, DnsAccountID: p.in.DnsAccountID,
		Issuer: issuerLabel(p.ca), IssueLog: out,
	}
	c.AltDomains = saveAltDomains(p.alts)
	if code != 0 {
		c.Status = "error"
		_ = s.db.Create(c).Error // 失败日志也入库便于排查
		return nil, errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "ACME 签发失败: "+firstLine(out))
	}
	if !p.in.AutoRenew {
		_, _ = s.run(ctx, fmt.Sprintf(acmeEnv+"%s --remove -d %s --ecc", acmeBin, p.main), 60)
	} else if err := s.reinstallCertHook(ctx, c); err != nil {
		return nil, err
	}
	s.probe(ctx, c)
	refreshCertStatus(c)
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	if logf != nil {
		logf("info", "签发完成：%s（颁发者 %s，到期 %s）", c.CertName, c.Issuer, c.NotAfter.Format("2006-01-02"))
	}
	return c, nil
}

// Issue ACME DNS 挑战签发，证书入库并部署到 nginx certs（同步阻塞，供内部复用）。
func (s *CertificateService) Issue(ctx context.Context, in CertIssueInput) (*model.Certificate, error) {
	p, err := s.prepareIssue(ctx, in)
	if err != nil {
		return nil, err
	}
	return s.issueExecute(ctx, nil, p)
}

// deployCertToSites 签发成功后把证书部署（绑定 HTTPS）到指定站点（多选，1Panel 同语义：
// 续签覆盖同名证书文件，站点 conf 无需变更）。逐站打点；部分失败任务标 failed（证书已入库的事实不变）。
func (s *CertificateService) deployCertToSites(ctx context.Context, logf TaskLogf, c *model.Certificate, siteIDs []uint) error {
	if len(siteIDs) == 0 {
		return nil
	}
	logf("info", "部署证书到 %d 个站点…", len(siteIDs))
	var failed []string
	for _, id := range siteIDs {
		var site model.Site
		if err := s.db.First(&site, id).Error; err != nil {
			failed = append(failed, fmt.Sprintf("#%d（站点不存在）", id))
			continue
		}
		if err := s.sites.ApplyCert(ctx, &site, c.ID, c.CertName); err != nil {
			failed = append(failed, fmt.Sprintf("%s（%v）", site.Name, err))
			logf("error", "站点 %s 部署失败: %v", site.Name, err)
			continue
		}
		logf("info", "已部署到站点 %s（%s）", site.Name, site.Domain)
	}
	if len(failed) > 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "证书已签发入库，但部分站点部署失败: "+strings.Join(failed, "；"))
	}
	return nil
}

// IssueTask 创建签发任务（返回任务 ID）：预备段同步快速失败，耗时执行进任务中心实时日志。
func (s *CertificateService) IssueTask(ctx context.Context, in CertIssueInput) (map[string]any, error) {
	p, err := s.prepareIssue(ctx, in)
	if err != nil {
		return nil, err
	}
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	if s.tasks.HasRunning([]string{TaskCertIssue}, p.main) {
		return nil, errs.Wrap(errs.ErrBadRequest, "该域名已有签发任务运行中，请稍后再试")
	}
	task, err := s.tasks.StartTask(TaskCertIssue, "签发证书 "+p.main, p.main, 15*time.Minute,
		func(ctx context.Context, logf TaskLogf) error {
			cert, err := s.issueExecute(ctx, logf, p)
			if err != nil {
				return err
			}
			return s.deployCertToSites(ctx, logf, cert, in.DeploySiteIDs)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID}, nil
}

// issuerLabel CA server → 展示名。
func issuerLabel(ca string) string {
	switch ca {
	case "zerossl":
		return "ZeroSSL"
	case "buypass":
		return "Buypass"
	default:
		return "Let's Encrypt"
	}
}

// CertUploadInput 上传输入（证书/私钥为 PEM 文本）。
type CertUploadInput struct {
	CertName string `json:"certName"` // 可选，默认主域名
	Domain   string `json:"domain"`
	Remark   string `json:"remark"`
	CertPEM  string `json:"certPem"`
	KeyPEM   string `json:"keyPem"`
}

// Upload 手动上传证书。
func (s *CertificateService) Upload(ctx context.Context, in CertUploadInput) (*model.Certificate, error) {
	main := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(in.Domain, "*.")))
	if !domainPattern.MatchString(main) {
		return nil, errs.Wrap(errs.ErrBadRequest, "主域名不合法: "+in.Domain)
	}
	certName := strings.TrimSpace(in.CertName)
	if certName == "" {
		certName = main
	}
	if !certNamePattern.MatchString(certName) {
		return nil, errs.Wrap(errs.ErrBadRequest, "证书名称不合法（字母/数字/点/中划线）")
	}
	if strings.TrimSpace(in.CertPEM) == "" || strings.TrimSpace(in.KeyPEM) == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "证书与私钥内容均不能为空")
	}
	if !strings.Contains(in.CertPEM, "BEGIN CERTIFICATE") || !strings.Contains(in.KeyPEM, "PRIVATE KEY") {
		return nil, errs.Wrap(errs.ErrBadRequest, "证书需为 PEM 格式（-----BEGIN CERTIFICATE-----），私钥需含 -----BEGIN ... PRIVATE KEY-----")
	}
	if _, err := s.byName(certName); err == nil {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "同名证书已存在")
	}

	// 公钥匹配校验（x509 pubkey md5 vs 私钥 pubout md5）
	crtTmp, keyTmp := path.Join("/tmp", "yp-crt-"+certName), path.Join("/tmp", "yp-key-"+certName)
	if err := s.writeViaFiles(ctx, crtTmp, in.CertPEM); err != nil {
		return nil, err
	}
	if err := s.writeViaFiles(ctx, keyTmp, in.KeyPEM); err != nil {
		return nil, err
	}
	defer func() {
		_, _ = s.run(ctx, fmt.Sprintf("rm -f %s %s", shQuote(crtTmp), shQuote(keyTmp)), 15)
	}()
	out, err := s.run(ctx, fmt.Sprintf("openssl x509 -in %s -noout -pubkey | openssl md5; openssl pkey -in %s -pubout | openssl md5",
		shQuote(crtTmp), shQuote(keyTmp)), 30)
	if err != nil || out.ExitCode != 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "证书或私钥解析失败，请检查 PEM 内容: "+firstLine(out.Output))
	}
	lines := strings.Fields(out.Output)
	vals := []string{}
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) == 32 && !strings.Contains(l, "=") && !strings.Contains(l, "-") {
			vals = append(vals, l)
		}
	}
	if len(vals) < 2 || vals[0] != vals[1] {
		return nil, errs.Wrap(errs.ErrBadRequest, "证书与私钥不匹配")
	}

	// 部署到 certs 卷
	keyPath := path.Join(nginxCertDir, certName+".key")
	crtPath := path.Join(nginxCertDir, certName+".crt")
	if err := s.writeViaFiles(ctx, crtPath, in.CertPEM); err != nil {
		return nil, err
	}
	if err := s.writeViaFiles(ctx, keyPath, in.KeyPEM); err != nil {
		return nil, err
	}
	c := &model.Certificate{
		CertName: certName, Domain: main, Provider: "upload",
		Remark: strings.TrimSpace(in.Remark), AutoRenew: false,
	}
	s.probe(ctx, c)
	refreshCertStatus(c)
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

// CertSelfSignedInput 自签输入。
type CertSelfSignedInput struct {
	Domain string `json:"domain"`
	Remark string `json:"remark"`
	Days   int    `json:"days"`
}

// SelfSigned 自签证书入库。
func (s *CertificateService) SelfSigned(ctx context.Context, in CertSelfSignedInput) (*model.Certificate, error) {
	main := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(in.Domain, "*.")))
	if !domainPattern.MatchString(main) {
		return nil, errs.Wrap(errs.ErrBadRequest, "域名不合法: "+in.Domain)
	}
	if _, err := s.byName(main); err == nil {
		return nil, errs.New(errs.CodeConflict, "error.conflict", "同名证书已存在")
	}
	days := in.Days
	if days < 1 || days > 3650 {
		days = 365
	}
	key := path.Join(nginxCertDir, main+".key")
	crt := path.Join(nginxCertDir, main+".crt")
	cmd := fmt.Sprintf("openssl req -x509 -nodes -newkey rsa:2048 -days %d -keyout %s -out %s -subj '/CN=%s'", days, key, crt, main)
	out, err := s.run(ctx, cmd, 60)
	if err != nil {
		return nil, err
	}
	if out.ExitCode != 0 {
		return nil, errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "自签失败: "+firstLine(out.Output))
	}
	c := &model.Certificate{
		CertName: main, Domain: main, Provider: "selfsigned",
		Remark: strings.TrimSpace(in.Remark), AutoRenew: false, Issuer: "YPanel Self-Signed",
	}
	s.probe(ctx, c)
	refreshCertStatus(c)
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CertificateService) byName(name string) (*model.Certificate, error) {
	var c model.Certificate
	if err := s.db.Where("cert_name = ?", name).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *CertificateService) writeViaFiles(ctx context.Context, p, content string) error {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return err
	}
	ac := agentclient.New(node.BaseURL, node.Token)
	_, err = agentclient.DoJSON[dto.FileWriteReq, struct{}](ac, ctx, "POST", "/agent/v1/files/write",
		&dto.FileWriteReq{Path: p, Content: content})
	return err
}

// ---- 环境变量兼容（systemd 注入路径）----

// SetEnvGetter 注入环境读取（main.go 装配；未注入则忽略环境变量 fallback）。
func (s *CertificateService) SetEnvGetter(f func(string) string) { s.envf = f }

func (s *CertificateService) env(key string) string {
	if s.envf != nil {
		return s.envf(key)
	}
	return ""
}

func (s *CertificateService) envEmail() string {
	if e := s.env("YPANEL_ACME_EMAIL"); e != "" {
		return e
	}
	return "admin@example.com"
}

func (s *CertificateService) envAliCreds() (key, secret string, ok bool) {
	key = s.env("YPANEL_ACME_ALI_KEY")
	secret = s.env("YPANEL_ACME_ALI_SECRET")
	return key, secret, key != "" && secret != ""
}

// ---- DNS 账户 ----

// DnsAccountItem 列表视图（密钥脱敏）。
type DnsAccountItem struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	AccessKey string `json:"accessKey"`
	CreatedAt time.Time `json:"createdAt"`
}

// ListDnsAccounts DNS 账户列表。
func (s *CertificateService) ListDnsAccounts() ([]DnsAccountItem, error) {
	var rows []model.DnsAccount
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DnsAccountItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, DnsAccountItem{ID: r.ID, Name: r.Name, Provider: r.Provider, AccessKey: r.AccessKey, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// DnsAccountInput DNS 账户输入。
type DnsAccountInput struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"` // aliyun / dnspod / cloudflare
	AccessKey string `json:"accessKey"`
	Secret    string `json:"secret"`
}

// CreateDnsAccount 创建 DNS 账户。
func (s *CertificateService) CreateDnsAccount(in DnsAccountInput) (*model.DnsAccount, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.AccessKey = strings.TrimSpace(in.AccessKey)
	if in.Name == "" || len(in.Name) > 64 {
		return nil, errs.Wrap(errs.ErrBadRequest, "账户名称不合法")
	}
	if in.Provider != "aliyun" && in.Provider != "dnspod" && in.Provider != "cloudflare" {
		return nil, errs.Wrap(errs.ErrBadRequest, "服务商仅支持 aliyun / dnspod / cloudflare")
	}
	if in.AccessKey == "" || in.Secret == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "AccessKey 与 Secret 均不能为空")
	}
	enc, err := s.encryptSecret(in.Secret)
	if err != nil {
		return nil, err
	}
	acc := &model.DnsAccount{Name: in.Name, Provider: in.Provider, AccessKey: in.AccessKey, SecretEnc: enc}
	if err := s.db.Create(acc).Error; err != nil {
		return nil, err
	}
	return acc, nil
}

// UpdateDnsAccount 更新 DNS 账户（Secret 留空保持不变）。
func (s *CertificateService) UpdateDnsAccount(id uint, in DnsAccountInput) error {
	var acc model.DnsAccount
	if err := s.db.First(&acc, id).Error; err != nil {
		return errs.New(errs.CodeNotFound, "error.notFound", "DNS 账户不存在")
	}
	if n := strings.TrimSpace(in.Name); n != "" {
		acc.Name = n
	}
	if p := strings.TrimSpace(in.Provider); p != "" {
		if p != "aliyun" && p != "dnspod" && p != "cloudflare" {
			return errs.Wrap(errs.ErrBadRequest, "服务商不合法")
		}
		acc.Provider = p
	}
	if k := strings.TrimSpace(in.AccessKey); k != "" {
		acc.AccessKey = k
	}
	if strings.TrimSpace(in.Secret) != "" {
		enc, err := s.encryptSecret(strings.TrimSpace(in.Secret))
		if err != nil {
			return err
		}
		acc.SecretEnc = enc
	}
	return s.db.Save(&acc).Error
}

// DeleteDnsAccount 删除 DNS 账户（被证书引用时拒绝）。
func (s *CertificateService) DeleteDnsAccount(id uint) error {
	var n int64
	_ = s.db.Model(&model.Certificate{}).Where("dns_account_id = ?", id).Count(&n).Error
	if n > 0 {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("仍有 %d 张证书使用该 DNS 账户（续签需要），请先解绑", n))
	}
	return s.db.Delete(&model.DnsAccount{}, id).Error
}

// ---- Acme 账户 ----

// ListAcmeAccounts ACME 账户列表。
func (s *CertificateService) ListAcmeAccounts() ([]model.AcmeAccount, error) {
	var rows []model.AcmeAccount
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// AcmeAccountInput ACME 账户输入。
type AcmeAccountInput struct {
	Email   string `json:"email"`
	CAType  string `json:"caType"`
	KeyType string `json:"keyType"`
}

// prepareAcmeAccount ACME 账户输入校验并建库记录（快速段，同步执行）。
func (s *CertificateService) prepareAcmeAccount(in AcmeAccountInput) (*model.AcmeAccount, error) {
	in.Email = strings.TrimSpace(in.Email)
	if !strings.Contains(in.Email, "@") {
		return nil, errs.Wrap(errs.ErrBadRequest, "邮箱不合法")
	}
	switch in.CAType {
	case "letsencrypt", "zerossl", "buypass":
	default:
		return nil, errs.Wrap(errs.ErrBadRequest, "CA 仅支持 letsencrypt / zerossl / buypass")
	}
	if in.KeyType == "" {
		in.KeyType = "ec-256"
	}
	switch in.KeyType {
	case "ec-256", "ec-384", "rsa-2048", "rsa-3072", "rsa-4096":
	default:
		return nil, errs.Wrap(errs.ErrBadRequest, "密钥算法不合法")
	}
	acc := &model.AcmeAccount{Email: in.Email, CAType: in.CAType, KeyType: in.KeyType}
	if err := s.db.Create(acc).Error; err != nil {
		return nil, err
	}
	return acc, nil
}

// registerAcme 向 CA 预注册账户（acme.sh --register-account，输出全程记日志）。
// 任务版失败即任务 failed（账户记录已入库，签发时会再次注册的兜底语义不变）。
func (s *CertificateService) registerAcme(ctx context.Context, logf TaskLogf, email, caType, keyType string) error {
	if err := s.ensureInstalled(ctx, logf, email); err != nil {
		return err
	}
	if logf != nil {
		logf("info", "注册 ACME 账户到 %s（%s）…", caType, email)
	}
	out, code, err := s.runStream(ctx, acmeEnv+fmt.Sprintf("%s --register-account -m %s --server %s --keylength %s",
		acmeBin, shQuote(email), caServer(caType), keyType), 180, logf)
	if err != nil {
		return err
	}
	if code != 0 {
		return errs.New(errs.CodeFileOpFailed, "error.fileOpFailed", "ACME 账户注册失败（签发时会自动重试）: "+firstLine(out))
	}
	if logf != nil {
		logf("info", "ACME 账户注册完成")
	}
	return nil
}

// CreateAcmeAccount 创建 ACME 账户并注册（返回任务 ID；注册过程与结果经任务中心日志可查）。
func (s *CertificateService) CreateAcmeAccount(ctx context.Context, in AcmeAccountInput) (map[string]any, error) {
	acc, err := s.prepareAcmeAccount(in)
	if err != nil {
		return nil, err
	}
	if s.tasks == nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "任务服务不可用")
	}
	ref := in.Email + ":" + in.CAType
	if s.tasks.HasRunning([]string{TaskAcmeRegister}, ref) {
		return nil, errs.Wrap(errs.ErrBadRequest, "该邮箱已有注册任务运行中，请稍后再试")
	}
	task, err := s.tasks.StartTask(TaskAcmeRegister,
		fmt.Sprintf("注册 ACME 账户（%s %s）", issuerLabel(caServer(in.CAType)), in.Email), ref, 6*time.Minute,
		func(ctx context.Context, logf TaskLogf) error {
			return s.registerAcme(ctx, logf, acc.Email, acc.CAType, acc.KeyType)
		})
	if err != nil {
		return nil, err
	}
	return map[string]any{"taskId": task.ID, "accountId": acc.ID}, nil
}

// DeleteAcmeAccount 删除 ACME 账户（被证书引用时拒绝）。
func (s *CertificateService) DeleteAcmeAccount(id uint) error {
	var n int64
	_ = s.db.Model(&model.Certificate{}).Where("acme_account_id = ?", id).Count(&n).Error
	if n > 0 {
		return errs.Wrap(errs.ErrBadRequest, fmt.Sprintf("仍有 %d 张证书使用该 ACME 账户（续签需要），请先解绑", n))
	}
	return s.db.Delete(&model.AcmeAccount{}, id).Error
}
