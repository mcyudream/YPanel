// DockerEnvService 多 Docker 环境（M35）：tcp 环境经 core 侧 moby client 直连远程 dockerd。
// local 环境走既有 agent dockerx 全量通道，本服务只承载 tcp 环境的核心子集与 CRUD。
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
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/moby/moby/client"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// defaultHTTPClient 通用出站 HTTP（registry 检查/agent 探活；超时各调用自行 ctx 控）。
var defaultHTTPClient = &http.Client{}

// envNamePattern 环境名白名单。
var envNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{1,31}$`)

// envHostPattern tcp 端点（host:port）。
var envHostPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+:\d{1,5}$`)

// DockerEnvService 多 Docker 环境。
type DockerEnvService struct {
	db     *gorm.DB
	nodes  *NodeService
	aesKey []byte
}

// NewDockerEnvService 创建（TLS 私钥加密密钥由 JWT 派生）。
func NewDockerEnvService(db *gorm.DB, nodes *NodeService, jwtSecret string) *DockerEnvService {
	sum := sha256.Sum256([]byte("ypanel-dockertls:" + jwtSecret))
	return &DockerEnvService{db: db, nodes: nodes, aesKey: sum[:]}
}

func (s *DockerEnvService) encryptKey(plain string) (string, error) {
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
	return base64.StdEncoding.EncodeToString(append(nonce, gcm.Seal(nil, nonce, []byte(plain), nil)...)), nil
}

func (s *DockerEnvService) decryptKey(enc string) (string, error) {
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

// EnvInput 环境入参。
type EnvInput struct {
	Name      string
	Type      string // local / tcp
	Endpoint  string
	TLSEnable bool
	TLSCA     string
	TLSCert   string
	TLSKey    string // 留空 = 保留原值（仅 Update）
	Remark    string
}

func validateEnvInput(in *EnvInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if !envNamePattern.MatchString(in.Name) {
		return errs.Wrap(errs.ErrBadRequest, "环境名不合法（字母开头，字母/数字/下划线/中划线，3-32 位）")
	}
	if in.Type != "local" && in.Type != "tcp" {
		return errs.Wrap(errs.ErrBadRequest, "不支持的类型: "+in.Type)
	}
	if in.Type == "tcp" {
		in.Endpoint = strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(in.Endpoint), "/"), "tcp://")
		if !envHostPattern.MatchString(in.Endpoint) {
			return errs.Wrap(errs.ErrBadRequest, "端点需为 host:port 形式")
		}
		if in.TLSEnable && in.TLSKey == "" {
			// 新建时 TLS 必须带客户端证书三件套（CA 可空=系统根）
			return errs.Wrap(errs.ErrBadRequest, "启用 TLS 需提供客户端证书与私钥")
		}
	}
	return nil
}

// List 环境列表。
func (s *DockerEnvService) List() ([]map[string]any, error) {
	var rows []model.DockerEnvironment
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	out = append(out, map[string]any{
		"id": 0, "name": "local", "type": "local", "endpoint": "", "tlsEnable": false,
		"remark": "本机（agent 全量能力）", "builtin": true,
	})
	for _, r := range rows {
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "endpoint": r.Endpoint,
			"tlsEnable": r.TLSEnable, "remark": r.Remark,
			"hasTLSKey": r.TLSKeyEnc != "", "createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// ByID 查环境。
func (s *DockerEnvService) ByID(id uint) (*model.DockerEnvironment, error) {
	var env model.DockerEnvironment
	if err := s.db.First(&env, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.envNotFound", "环境不存在")
	}
	return &env, nil
}

// Create 新建环境（保存前连通性验证）。
func (s *DockerEnvService) Create(ctx context.Context, in EnvInput) (*model.DockerEnvironment, error) {
	if err := validateEnvInput(&in); err != nil {
		return nil, err
	}
	var count int64
	_ = s.db.Model(&model.DockerEnvironment{}).Where("name = ?", in.Name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.envExists", "环境名已存在")
	}
	row := &model.DockerEnvironment{
		Name: in.Name, Type: in.Type, Endpoint: in.Endpoint, TLSEnable: in.TLSEnable,
		TLSCA: in.TLSCA, TLSCert: in.TLSCert, Remark: in.Remark,
	}
	if in.TLSKey != "" {
		enc, err := s.encryptKey(in.TLSKey)
		if err != nil {
			return nil, err
		}
		row.TLSKeyEnc = enc
	}
	if in.Type == "tcp" {
		cli, closeFn, err := s.tcpClient(row, in.TLSKey)
		if err != nil {
			return nil, err
		}
		pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		_, perr := cli.Ping(pctx, client.PingOptions{})
		cancel()
		closeFn()
		if perr != nil {
			return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接测试失败: "+perr.Error())
		}
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Update 更新环境（TLS 私钥留空保留原值；变更后回读验证）。
func (s *DockerEnvService) Update(ctx context.Context, id uint, in EnvInput) error {
	row, err := s.ByID(id)
	if err != nil {
		return err
	}
	if err := validateEnvInput(&in); err != nil {
		return err
	}
	if in.TLSEnable && in.TLSKey == "" && row.TLSKeyEnc == "" {
		return errs.Wrap(errs.ErrBadRequest, "启用 TLS 需提供客户端证书与私钥")
	}
	updates := map[string]any{
		"name": in.Name, "type": in.Type, "endpoint": in.Endpoint,
		"tls_enable": in.TLSEnable, "tls_ca": in.TLSCA, "tls_cert": in.TLSCert, "remark": in.Remark,
	}
	if in.TLSKey != "" {
		enc, err := s.encryptKey(in.TLSKey)
		if err != nil {
			return err
		}
		updates["tls_key_enc"] = enc
	}
	if err := s.db.Model(row).Updates(updates).Error; err != nil {
		return err
	}
	fresh, err := s.ByID(id)
	if err != nil {
		return err
	}
	if fresh.Type == "tcp" {
		cli, closeFn, err := s.tcpClientRow(ctx, fresh)
		if err != nil {
			return err
		}
		pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		_, perr := cli.Ping(pctx, client.PingOptions{})
		cancel()
		closeFn()
		if perr != nil {
			return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接测试失败: "+perr.Error())
		}
	}
	return nil
}

// Delete 删除环境。
func (s *DockerEnvService) Delete(id uint) error {
	return s.db.Delete(&model.DockerEnvironment{}, id).Error
}

// Test 环境连通性测试。
func (s *DockerEnvService) Test(ctx context.Context, id uint) (map[string]any, error) {
	start := time.Now()
	if id == 0 {
		// local 环境走 agent 探活
		ac, err := s.nodesClient()
		if err != nil {
			return nil, err
		}
		if _, err := agentGetJSONRetry(ac, ctx); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "latencyMs": time.Since(start).Milliseconds()}, nil
	}
	env, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	cli, closeFn, err := s.tcpClientRow(ctx, env)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	ping, perr := cli.Ping(pctx, client.PingOptions{})
	if perr != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接失败: "+perr.Error())
	}
	out := map[string]any{"ok": true, "latencyMs": time.Since(start).Milliseconds()}
	if ping.APIVersion != "" {
		out["apiVersion"] = ping.APIVersion
	}
	return out, nil
}

// tcpClientRow 按库里行构造客户端（解密私钥写临时文件，用完即删）。
func (s *DockerEnvService) tcpClientRow(ctx context.Context, env *model.DockerEnvironment) (*client.Client, func(), error) {
	key := ""
	if env.TLSKeyEnc != "" {
		var err error
		key, err = s.decryptKey(env.TLSKeyEnc)
		if err != nil {
			return nil, func() {}, err
		}
	}
	return s.tcpClient(env, key)
}

// tcpClient 构造 moby 客户端（TLS 凭据经临时文件注入，返回的 closeFn 负责清理）。
func (s *DockerEnvService) tcpClient(env *model.DockerEnvironment, keyPEM string) (*client.Client, func(), error) {
	opts := []client.Opt{
		client.WithHost("tcp://" + env.Endpoint),
		client.WithAPIVersionNegotiation(),
	}
	var tmpFiles []string
	closeFn := func() {
		for _, f := range tmpFiles {
			_ = os.Remove(f)
		}
	}
	if env.TLSEnable {
		writeTmp := func(content string) (string, error) {
			f, err := os.CreateTemp("", "yp-dockertls-*")
			if err != nil {
				return "", err
			}
			if _, err := f.WriteString(content); err != nil {
				_ = f.Close()
				return "", err
			}
			_ = f.Close()
			_ = os.Chmod(f.Name(), 0o600)
			tmpFiles = append(tmpFiles, f.Name())
			return f.Name(), nil
		}
		caFile := ""
		if env.TLSCA != "" {
			var err error
			if caFile, err = writeTmp(env.TLSCA); err != nil {
				closeFn()
				return nil, closeFn, err
			}
		}
		certFile := ""
		keyFile := ""
		if env.TLSCert != "" && keyPEM != "" {
			var err error
			if certFile, err = writeTmp(env.TLSCert); err != nil {
				closeFn()
				return nil, closeFn, err
			}
			if keyFile, err = writeTmp(keyPEM); err != nil {
				closeFn()
				return nil, closeFn, err
			}
		}
		opts = append(opts, client.WithTLSClientConfig(caFile, certFile, keyFile))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		closeFn()
		return nil, closeFn, errs.Wrap(errs.ErrBadRequest, "docker 客户端构造失败: "+err.Error())
	}
	return cli, closeFn, nil
}

func (s *DockerEnvService) nodesClient() (*nodeClientPair, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return &nodeClientPair{base: node.BaseURL, token: node.Token}, nil
}

type nodeClientPair struct {
	base  string
	token string
}

// agentGetJSONRetry local 环境探活（health 端点直连）。
func agentGetJSONRetry(ac *nodeClientPair, ctx context.Context) (map[string]any, error) {
	req, err := httpRequest(ctx, "GET", ac.base+"/agent/v1/health", ac.token, nil)
	if err != nil {
		return nil, err
	}
	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接失败: "+err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	var env struct {
		Code int `json:"code"`
		Data map[string]any
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "响应解析失败: "+err.Error())
	}
	if env.Code != 0 {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "agent 健康检查失败")
	}
	return env.Data, nil
}

// EnvContainers tcp 环境容器列表（精简视图）。
func (s *DockerEnvService) EnvContainers(ctx context.Context, id uint) ([]map[string]any, error) {
	env, err := s.envTCP(ctx, id)
	if err != nil {
		return nil, err
	}
	cli, closeFn, err := s.tcpClientRow(ctx, env)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	actx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	res, err := cli.ContainerList(actx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "容器列表失败: "+err.Error())
	}
	out := make([]map[string]any, 0, len(res.Items))
	for _, c := range res.Items {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		img := ""
		if len(c.Image) > 0 {
			img = c.Image
		}
		ports := []string{}
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				proto := string(p.Type)
				ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublicPort, uint16(p.PrivatePort), proto))
			}
		}
		out = append(out, map[string]any{
			"id": c.ID[:12], "name": name, "image": img, "state": string(c.State),
			"status": c.Status, "ports": ports,
		})
	}
	return out, nil
}

// EnvImages tcp 环境镜像列表（精简视图）。
func (s *DockerEnvService) EnvImages(ctx context.Context, id uint) ([]map[string]any, error) {
	env, err := s.envTCP(ctx, id)
	if err != nil {
		return nil, err
	}
	cli, closeFn, err := s.tcpClientRow(ctx, env)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	actx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	res, err := cli.ImageList(actx, client.ImageListOptions{All: false})
	if err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "镜像列表失败: "+err.Error())
	}
	out := make([]map[string]any, 0, len(res.Items))
	for _, im := range res.Items {
		sizeMB := float64(im.Size) / 1024 / 1024
		out = append(out, map[string]any{
			"id": im.ID[7:19], "tags": im.RepoTags, "sizeMb": sizeMB, "created": im.Created,
		})
	}
	return out, nil
}

// EnvContainerAction tcp 环境容器动作（start/stop/restart/remove）。
func (s *DockerEnvService) EnvContainerAction(ctx context.Context, id uint, name, action string) error {
	env, err := s.envTCP(ctx, id)
	if err != nil {
		return err
	}
	if !containerNamePattern.MatchString(name) {
		return errs.Wrap(errs.ErrBadRequest, "容器名不合法")
	}
	cli, closeFn, err := s.tcpClientRow(ctx, env)
	if err != nil {
		return err
	}
	defer closeFn()
	actx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	switch action {
	case "start":
		_, err = cli.ContainerStart(actx, name, client.ContainerStartOptions{})
	case "stop":
		_, err = cli.ContainerStop(actx, name, client.ContainerStopOptions{})
	case "restart":
		_, err = cli.ContainerRestart(actx, name, client.ContainerRestartOptions{})
	case "remove":
		_, err = cli.ContainerRemove(actx, name, client.ContainerRemoveOptions{Force: true})
	default:
		return errs.Wrap(errs.ErrBadRequest, "不支持的动作: "+action)
	}
	if err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "操作失败: "+err.Error())
	}
	return nil
}

// EnvImageRemove tcp 环境镜像删除。
func (s *DockerEnvService) EnvImageRemove(ctx context.Context, id uint, imageID string, force bool) error {
	env, err := s.envTCP(ctx, id)
	if err != nil {
		return err
	}
	if !imageRefPattern.MatchString(imageID) {
		return errs.Wrap(errs.ErrBadRequest, "镜像 ID 不合法")
	}
	cli, closeFn, err := s.tcpClientRow(ctx, env)
	if err != nil {
		return err
	}
	defer closeFn()
	actx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	_, err = cli.ImageRemove(actx, imageID, client.ImageRemoveOptions{Force: force})
	if err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "删除失败: "+err.Error())
	}
	return nil
}

func (s *DockerEnvService) envTCP(ctx context.Context, id uint) (*model.DockerEnvironment, error) {
	if id == 0 {
		return nil, errs.Wrap(errs.ErrBadRequest, "本机环境请使用容器/镜像页（全量能力）")
	}
	env, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	if env.Type != "tcp" {
		return nil, errs.Wrap(errs.ErrBadRequest, "仅 tcp 环境支持远程操作")
	}
	return env, nil
}

// httpRequest 构造带可选 token 的请求（agentclient 之外的轻量通道）。
func httpRequest(ctx context.Context, method, url, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}
