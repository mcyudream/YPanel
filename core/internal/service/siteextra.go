// 站点配置域第二批（对标 1Panel）：防盗链/Basic 认证/CORS/重定向/真实 IP/连接限制/负载均衡。
// 单列 JSON（sites.conf_json）聚合存储；每个域独立 GET/PUT，PUT 统一走
// "校验 → 落库 → writeConf(nginx -t 失败回滚)"。Basic 认证 htpasswd 由容器内 openssl 生成写入
// conf.d/<site>.htpasswd（非 .conf 后缀不会被 nginx include，无需给容器加挂载）。
package service

import (
	"context"
	cryptoRand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// ---- 域结构 ----

// SiteAntiLeech 防盗链。
type SiteAntiLeech struct {
	Enable        bool     `json:"enable"`
	ValidReferers []string `json:"validReferers"` // 除 none/blocked/server_names 外的白名单 referer
	AllowNone     bool     `json:"allowNone"`     // 允许空 referer
	AllowBlocked  bool     `json:"allowBlocked"`  // 允许被防火墙遮蔽的 referer
	ReturnCode    int      `json:"returnCode"`    // 403 / 404
}

// SiteAuthBasicUser Basic 认证用户。
type SiteAuthBasicUser struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// SiteAuthBasic Basic 认证。
type SiteAuthBasic struct {
	Enable bool                `json:"enable"`
	Realm  string              `json:"realm"`
	Users  []SiteAuthBasicUser `json:"users"`
}

// SiteCORS 跨域配置。
type SiteCORS struct {
	Enable           bool     `json:"enable"`
	AllowOrigins     []string `json:"allowOrigins"` // * 或 scheme://host
	AllowMethods     []string `json:"allowMethods"`
	AllowHeaders     []string `json:"allowHeaders"`
	AllowCredentials bool     `json:"allowCredentials"`
	MaxAge           int      `json:"maxAge"` // 秒
}

// SiteRedirect 整站重定向。
type SiteRedirect struct {
	Enable bool   `json:"enable"`
	Target string `json:"target"` // https://new.example.com（可含路径）
	Code   int    `json:"code"`   // 301/302/307/308
}

// SiteRealIP 真实 IP。
type SiteRealIP struct {
	Enable        bool     `json:"enable"`
	TrustedProxies []string `json:"trustedProxies"` // IP 或 CIDR
	Header        string   `json:"header"`         // X-Forwarded-For / X-Real-IP / CF-Connecting-IP
}

// SiteLimitConn 连接限制。
type SiteLimitConn struct {
	Enable    bool `json:"enable"`
	ConnPerIP int  `json:"connPerIP"`
}

// SiteUpstream 负载均衡上游。
type SiteUpstream struct {
	Address string `json:"address"` // host:port
	Weight  int    `json:"weight"`
}

// SiteLoadBalance 负载均衡（仅 proxy 类型）。
type SiteLoadBalance struct {
	Enable    bool           `json:"enable"`
	Strategy  string         `json:"strategy"` // round-robin / least_conn / ip_hash
	Upstreams []SiteUpstream `json:"upstreams"`
}

// SiteExtraConf 第二批配置域聚合。
type SiteExtraConf struct {
	AntiLeech   *SiteAntiLeech   `json:"antiLeech,omitempty"`
	AuthBasic   *SiteAuthBasic   `json:"authBasic,omitempty"`
	CORS        *SiteCORS        `json:"cors,omitempty"`
	Redirect    *SiteRedirect    `json:"redirect,omitempty"`
	RealIP      *SiteRealIP     `json:"realIP,omitempty"`
	LimitConn   *SiteLimitConn   `json:"limitConn,omitempty"`
	LoadBalance *SiteLoadBalance `json:"loadBalance,omitempty"`
}

// parseSiteExtra 解析站点额外配置域。
func parseSiteExtra(site *model.Site) SiteExtraConf {
	var out SiteExtraConf
	if site.ConfJSON != "" {
		_ = json.Unmarshal([]byte(site.ConfJSON), &out)
	}
	return out
}

// saveExtra 序列化并落库（不触发 writeConf，由调用方按需调用）。
func (s *SiteService) saveExtra(site *model.Site, extra SiteExtraConf) error {
	raw, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	return s.db.Model(site).Update("conf_json", string(raw)).Error
}

// getExtraForWrite 取站点 + 解析域配置。
func (s *SiteService) getExtraForWrite(id uint) (*model.Site, SiteExtraConf, error) {
	site, err := s.siteByID(id)
	if err != nil {
		return nil, SiteExtraConf{}, err
	}
	s = s.forSite(site)
	return site, parseSiteExtra(site), nil
}

// applyExtra 落库并重写配置。
func (s *SiteService) applyExtra(site *model.Site, extra SiteExtraConf) error {
	if err := s.saveExtra(site, extra); err != nil {
		return err
	}
	return s.writeConf(context.Background(), site, confTemplate(site, site.CertDomain != "", parseWaf(site)))
}

// ---- 防盗链 ----

// GetAntiLeech 读取。
func (s *SiteService) GetAntiLeech(id uint) (SiteAntiLeech, error) {
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteAntiLeech{}, err
	}
	if extra.AntiLeech == nil {
		return SiteAntiLeech{ReturnCode: 403}, nil
	}
	_ = site
	return *extra.AntiLeech, nil
}

// UpdateAntiLeech 更新防盗链。
func (s *SiteService) UpdateAntiLeech(ctx context.Context, id uint, conf SiteAntiLeech) (SiteAntiLeech, error) {
	if conf.ReturnCode != 403 && conf.ReturnCode != 404 {
		return SiteAntiLeech{}, errs.Wrap(errs.ErrBadRequest, "拦截返回码仅支持 403/404")
	}
	cleaned := make([]string, 0, len(conf.ValidReferers))
	for _, r := range conf.ValidReferers {
		if r = strings.TrimSpace(r); r != "" {
			cleaned = append(cleaned, r)
		}
	}
	conf.ValidReferers = cleaned
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteAntiLeech{}, err
	}
	extra.AntiLeech = &conf
	return conf, s.applyExtra(site, extra)
}

// ---- Basic 认证 ----

// GetAuthBasic 读取（密码不回显）。
func (s *SiteService) GetAuthBasic(id uint) (SiteAuthBasic, error) {
	_, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteAuthBasic{}, err
	}
	out := SiteAuthBasic{Realm: "Restricted"}
	if extra.AuthBasic != nil {
		out = *extra.AuthBasic
		out.Realm = orDefault(out.Realm, "Restricted")
	}
	// 密码不回显，仅保留用户名占位
	for i := range out.Users {
		out.Users[i].Password = ""
	}
	return out, nil
}

// UpdateAuthBasic 更新（密码留空表示沿用旧值）。
func (s *SiteService) UpdateAuthBasic(ctx context.Context, id uint, conf SiteAuthBasic) (SiteAuthBasic, error) {
	if conf.Enable && len(conf.Users) == 0 {
		return SiteAuthBasic{}, errs.Wrap(errs.ErrBadRequest, "启用 Basic 认证需至少一个用户")
	}
	if conf.Realm == "" {
		conf.Realm = "Restricted"
	}
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteAuthBasic{}, err
	}
	// 保留旧密码：提交为空的密码沿用旧值
	if extra.AuthBasic != nil {
		old := map[string]string{}
		for _, u := range extra.AuthBasic.Users {
			old[u.User] = u.Password
		}
		for i := range conf.Users {
			if conf.Users[i].Password == "" {
				conf.Users[i].Password = old[conf.Users[i].User]
			}
		}
	}
	seen := map[string]bool{}
	for _, u := range conf.Users {
		if u.User == "" || u.Password == "" {
			return SiteAuthBasic{}, errs.Wrap(errs.ErrBadRequest, "Basic 认证用户名与密码不能为空")
		}
		if strings.ContainsAny(u.User, ": \t") {
			return SiteAuthBasic{}, errs.Wrap(errs.ErrBadRequest, "用户名不能含空格或冒号: "+u.User)
		}
		if seen[u.User] {
			return SiteAuthBasic{}, errs.Wrap(errs.ErrBadRequest, "用户名重复: "+u.User)
		}
		seen[u.User] = true
	}
	if err := s.writeHtpasswd(ctx, site.Name, conf.Users); err != nil {
		return SiteAuthBasic{}, err
	}
	extra.AuthBasic = &conf
	if err := s.applyExtra(site, extra); err != nil {
		return SiteAuthBasic{}, err
	}
	return s.GetAuthBasic(id)
}

// writeHtpasswd 用容器内 openssl 生成 apr1 哈希并写 htpasswd 到宿主机 conf.d。
// 不手写 apr1：算法细节易错（字节重排/清零语义），openssl 与 nginx 天然兼容（同 selfSign 模式）。
func (s *SiteService) writeHtpasswd(ctx context.Context, siteName string, users []SiteAuthBasicUser) error {
	if len(users) == 0 {
		return nil
	}
	ac, err := s.client()
	if err != nil {
		return err
	}
	// 逐用户经 openssl 生成 apr1（密码走 base64 中转避免 shell 引号问题）
	var lines []string
	for _, u := range users {
		b64 := base64.StdEncoding.EncodeToString([]byte(u.Password))
		one := fmt.Sprintf("echo %s | base64 -d | openssl passwd -apr1 -salt %s -stdin", b64, randomSalt(8))
		out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
			&dto.ExecReq{Command: one, TimeoutSecs: 30})
		if err != nil {
			return err
		}
		if out.ExitCode != 0 || strings.TrimSpace(out.Output) == "" {
			return errs.Wrapc(errs.CodeFileOpFailed, "apr1 生成失败: "+firstLine(out.Output))
		}
		lines = append(lines, u.User+":"+strings.TrimSpace(out.Output))
	}
	// 单行 printf 写文件（\n 转义为字面序列）
	content := strings.ReplaceAll(strings.Join(lines, "\n")+"\n", "\n", "\\n")
	target := path.Join(nginxConfDir, siteName+".htpasswd")
	write := fmt.Sprintf("printf '%s' > %s", content, target)
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: write, TimeoutSecs: 30})
	if err != nil {
		return err
	}
	if out.ExitCode != 0 {
		return errs.Wrapc(errs.CodeFileOpFailed, "htpasswd 写入失败: "+firstLine(out.Output))
	}
	return nil
}

// ---- CORS ----

var corsMethodSet = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true}

// GetCORS 读取。
func (s *SiteService) GetCORS(id uint) (SiteCORS, error) {
	_, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteCORS{}, err
	}
	if extra.CORS == nil {
		return SiteCORS{AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Content-Type", "Authorization"}, MaxAge: 86400}, nil
	}
	return *extra.CORS, nil
}

// UpdateCORS 更新。
func (s *SiteService) UpdateCORS(ctx context.Context, id uint, conf SiteCORS) (SiteCORS, error) {
	for _, o := range conf.AllowOrigins {
		if o != "*" && !strings.HasPrefix(o, "http://") && !strings.HasPrefix(o, "https://") {
			return SiteCORS{}, errs.Wrap(errs.ErrBadRequest, "允许来源需为 * 或 http(s):// 开头: "+o)
		}
	}
	for _, m := range conf.AllowMethods {
		if !corsMethodSet[m] {
			return SiteCORS{}, errs.Wrap(errs.ErrBadRequest, "不支持的 HTTP 方法: "+m)
		}
	}
	if conf.MaxAge < 0 || conf.MaxAge > 86400*7 {
		return SiteCORS{}, errs.Wrap(errs.ErrBadRequest, "预检缓存时间需在 0-604800 秒")
	}
	if conf.Enable {
		if len(conf.AllowOrigins) == 0 {
			return SiteCORS{}, errs.Wrap(errs.ErrBadRequest, "启用 CORS 需至少一个允许来源")
		}
		if len(conf.AllowMethods) == 0 {
			conf.AllowMethods = []string{"GET", "POST", "OPTIONS"}
		}
	}
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteCORS{}, err
	}
	extra.CORS = &conf
	return conf, s.applyExtra(site, extra)
}

// ---- 重定向 ----

// GetRedirect 读取。
func (s *SiteService) GetRedirect(id uint) (SiteRedirect, error) {
	_, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteRedirect{}, err
	}
	if extra.Redirect == nil {
		return SiteRedirect{Code: 301}, nil
	}
	return *extra.Redirect, nil
}

// UpdateRedirect 更新。
func (s *SiteService) UpdateRedirect(ctx context.Context, id uint, conf SiteRedirect) (SiteRedirect, error) {
	if conf.Code != 301 && conf.Code != 302 && conf.Code != 307 && conf.Code != 308 {
		return SiteRedirect{}, errs.Wrap(errs.ErrBadRequest, "重定向码仅支持 301/302/307/308")
	}
	if conf.Enable && !strings.HasPrefix(conf.Target, "http://") && !strings.HasPrefix(conf.Target, "https://") {
		return SiteRedirect{}, errs.Wrap(errs.ErrBadRequest, "重定向目标需为 http(s):// 地址")
	}
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteRedirect{}, err
	}
	extra.Redirect = &conf
	return conf, s.applyExtra(site, extra)
}

// ---- 真实 IP ----

var realIPHeaders = map[string]bool{"X-Forwarded-For": true, "X-Real-IP": true, "CF-Connecting-IP": true}

// GetRealIP 读取。
func (s *SiteService) GetRealIP(id uint) (SiteRealIP, error) {
	_, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteRealIP{}, err
	}
	if extra.RealIP == nil {
		return SiteRealIP{Header: "X-Forwarded-For"}, nil
	}
	return *extra.RealIP, nil
}

// UpdateRealIP 更新。
func (s *SiteService) UpdateRealIP(ctx context.Context, id uint, conf SiteRealIP) (SiteRealIP, error) {
	if !realIPHeaders[conf.Header] {
		return SiteRealIP{}, errs.Wrap(errs.ErrBadRequest, "真实 IP 头仅支持 X-Forwarded-For / X-Real-IP / CF-Connecting-IP")
	}
	cleaned := make([]string, 0, len(conf.TrustedProxies))
	for _, p := range conf.TrustedProxies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !isAllowedIPFormat(strings.Split(p, "/")[0]) {
			return SiteRealIP{}, errs.Wrap(errs.ErrBadRequest, "可信代理地址不合法: "+p)
		}
		cleaned = append(cleaned, p)
	}
	conf.TrustedProxies = cleaned
	if conf.Enable && len(cleaned) == 0 {
		return SiteRealIP{}, errs.Wrap(errs.ErrBadRequest, "启用真实 IP 需至少一个可信代理")
	}
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteRealIP{}, err
	}
	extra.RealIP = &conf
	return conf, s.applyExtra(site, extra)
}

// ---- 连接限制 ----

// GetLimitConn 读取。
func (s *SiteService) GetLimitConn(id uint) (SiteLimitConn, error) {
	_, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteLimitConn{}, err
	}
	if extra.LimitConn == nil {
		return SiteLimitConn{ConnPerIP: 20}, nil
	}
	return *extra.LimitConn, nil
}

// UpdateLimitConn 更新。
func (s *SiteService) UpdateLimitConn(ctx context.Context, id uint, conf SiteLimitConn) (SiteLimitConn, error) {
	if conf.ConnPerIP < 1 || conf.ConnPerIP > 10000 {
		return SiteLimitConn{}, errs.Wrap(errs.ErrBadRequest, "单 IP 并发连接需在 1-10000")
	}
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteLimitConn{}, err
	}
	extra.LimitConn = &conf
	return conf, s.applyExtra(site, extra)
}

// ---- 负载均衡 ----

// GetLoadBalance 读取。
func (s *SiteService) GetLoadBalance(id uint) (SiteLoadBalance, error) {
	_, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteLoadBalance{}, err
	}
	if extra.LoadBalance == nil {
		return SiteLoadBalance{Strategy: "round-robin"}, nil
	}
	return *extra.LoadBalance, nil
}

// UpdateLoadBalance 更新。
func (s *SiteService) UpdateLoadBalance(ctx context.Context, id uint, conf SiteLoadBalance) (SiteLoadBalance, error) {
	switch conf.Strategy {
	case "round-robin":
		conf.Strategy = "" // nginx 默认即轮询
	case "least_conn", "ip_hash":
	default:
		return SiteLoadBalance{}, errs.Wrap(errs.ErrBadRequest, "均衡策略仅支持 round-robin/least_conn/ip_hash")
	}
	if conf.Enable {
		if len(conf.Upstreams) < 2 {
			return SiteLoadBalance{}, errs.Wrap(errs.ErrBadRequest, "负载均衡需至少两个上游节点")
		}
		for i := range conf.Upstreams {
			u := strings.TrimPrefix(strings.TrimPrefix(conf.Upstreams[i].Address, "http://"), "https://")
			if u == "" || strings.Contains(u, "/") {
				return SiteLoadBalance{}, errs.Wrap(errs.ErrBadRequest, "上游地址需为 host:port 形式: "+conf.Upstreams[i].Address)
			}
			if conf.Upstreams[i].Weight < 0 || conf.Upstreams[i].Weight > 100 {
				return SiteLoadBalance{}, errs.Wrap(errs.ErrBadRequest, "权重需在 0-100")
			}
		}
	}
	site, extra, err := s.getExtraForWrite(id)
	if err != nil {
		return SiteLoadBalance{}, err
	}
	if site.Type != "proxy" {
		return SiteLoadBalance{}, errs.Wrap(errs.ErrBadRequest, "负载均衡仅支持反向代理站点")
	}
	extra.LoadBalance = &conf
	out := conf
	if out.Strategy == "" {
		out.Strategy = "round-robin"
	}
	return out, s.applyExtra(site, extra)
}

// ---- 其他（自定义 location 迁移入口用 ext API，此处仅聚合读取） ----

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// ---- apr1（Apache MD5 crypt）----

const apr1Magic = "$apr1$"

const apr1Itoa64 = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func apr1To64(v uint, n int) string {
	out := make([]byte, n)
	for i := n - 1; i >= 0; i-- {
		out[i] = apr1Itoa64[v&0x3f]
		v >>= 6
	}
	return string(out)
}

func randomSalt(n int) string {
	const alphabet = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	_, _ = cryptoRand.Read(b)
	out := make([]byte, n)
	for i := range b {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out)
}

// quoteGo 生成带引号的 Go 字符串字面量（合成 nginx 指令时转义用）。
func quoteGo(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
