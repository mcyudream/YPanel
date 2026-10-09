// StorageService 远程备份存储（M34）：s3 兼容（S3/MinIO/OSS/COS）/ webdav / sftp 三客户端。
// 上传/拉回全程流式（agent 下载流 → 远端 Put；远端 Open → agent multipart 上传），不经临时盘。
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
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-webdav"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// storageTarget 统一远端目标接口（key 相对 BackupPath 根）。
type storageTarget interface {
	Test(ctx context.Context) error
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	List(ctx context.Context, prefix string) ([]storageObject, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

type storageObject struct {
	Name    string    `json:"name"` // 相对 BackupPath 的 key
	Size    int64     `json:"sizeMb"`
	ModTime time.Time `json:"modTime"`
}

// StorageAccountInput 账号入参（Secret 明文只进不出）。
type StorageAccountInput struct {
	Name       string
	Type       string // s3 / webdav / sftp
	Endpoint   string
	Region     string
	Bucket     string
	AccessKey  string
	Secret     string // 留空 = 保留原值（仅 Update）
	BackupPath string
	UseSSL     bool
	Remark     string
}

// StorageService 远程存储账号管理与备份上传。
type StorageService struct {
	db     *gorm.DB
	nodes  *NodeService
	aesKey []byte
}

// NewStorageService 创建（加密密钥由 JWT 密钥派生，同 DnsAccount 先例）。
func NewStorageService(db *gorm.DB, nodes *NodeService, jwtSecret string) *StorageService {
	sum := sha256.Sum256([]byte("ypanel-storagecred:" + jwtSecret))
	return &StorageService{db: db, nodes: nodes, aesKey: sum[:]}
}

func (s *StorageService) client() (*agentclient.Client, error) {
	node, err := s.nodes.ByID("local")
	if err != nil {
		return nil, err
	}
	return agentclient.New(node.BaseURL, node.Token), nil
}

func (s *StorageService) encryptSecret(plain string) (string, error) {
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

func (s *StorageService) decryptSecret(enc string) (string, error) {
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

// validate 入参校验（BackupPath 归一为无首尾斜杠的相对前缀，防穿越）。
func validateStorageInput(in *StorageAccountInput) error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return errs.Wrap(errs.ErrBadRequest, "名称不能为空")
	}
	switch in.Type {
	case "s3":
		if strings.TrimSpace(in.Bucket) == "" {
			return errs.Wrap(errs.ErrBadRequest, "s3 类型必须填写 Bucket")
		}
	case "webdav", "sftp":
	default:
		return errs.Wrap(errs.ErrBadRequest, "不支持的存储类型: "+in.Type)
	}
	if strings.TrimSpace(in.Endpoint) == "" {
		return errs.Wrap(errs.ErrBadRequest, "Endpoint 不能为空")
	}
	p := strings.TrimSpace(in.BackupPath)
	if p == "" {
		p = "ypanel-backups"
	}
	if strings.Contains(p, "..") {
		return errs.Wrap(errs.ErrBadRequest, "备份路径不合法")
	}
	in.BackupPath = strings.Trim(p, "/")
	return nil
}

// List 账号列表（凭据不回显）。
func (s *StorageService) List() ([]map[string]any, error) {
	var rows []model.StorageAccount
	if err := s.db.Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id": r.ID, "name": r.Name, "type": r.Type, "endpoint": r.Endpoint,
			"region": r.Region, "bucket": r.Bucket, "accessKey": r.AccessKey,
			"backupPath": r.BackupPath, "useSSL": r.UseSSL, "remark": r.Remark,
			"hasSecret": r.SecretEnc != "", "createdAt": r.CreatedAt,
		})
	}
	return out, nil
}

// ByID 查账号。
func (s *StorageService) ByID(id uint) (*model.StorageAccount, error) {
	var acct model.StorageAccount
	if err := s.db.First(&acct, id).Error; err != nil {
		return nil, errs.New(errs.CodeNotFound, "error.storageAccountNotFound", "存储账号不存在")
	}
	return &acct, nil
}

// Create 新建账号。
func (s *StorageService) Create(ctx context.Context, in StorageAccountInput) (*model.StorageAccount, error) {
	if err := validateStorageInput(&in); err != nil {
		return nil, err
	}
	if in.Secret == "" {
		return nil, errs.Wrap(errs.ErrBadRequest, "访问凭据不能为空")
	}
	var count int64
	_ = s.db.Model(&model.StorageAccount{}).Where("name = ?", in.Name).Count(&count).Error
	if count > 0 {
		return nil, errs.New(errs.CodeConflict, "error.storageAccountExists", "账号名已存在")
	}
	enc, err := s.encryptSecret(in.Secret)
	if err != nil {
		return nil, err
	}
	row := &model.StorageAccount{
		Name: in.Name, Type: in.Type, Endpoint: strings.TrimSpace(in.Endpoint), Region: strings.TrimSpace(in.Region),
		Bucket: strings.TrimSpace(in.Bucket), AccessKey: strings.TrimSpace(in.AccessKey), SecretEnc: enc,
		BackupPath: in.BackupPath, UseSSL: in.UseSSL, Remark: in.Remark,
	}
	// 保存前先连通性验证，避免存入不可用账号
	tgt, err := s.targetFor(row, in.Secret)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tgt.Close() }()
	if err := tgt.Test(ctx); err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接测试失败: "+err.Error())
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

// Update 更新账号（Secret 留空保留原值）。
func (s *StorageService) Update(ctx context.Context, id uint, in StorageAccountInput) error {
	row, err := s.ByID(id)
	if err != nil {
		return err
	}
	if err := validateStorageInput(&in); err != nil {
		return err
	}
	secret := in.Secret
	if secret == "" {
		secret, err = s.decryptSecret(row.SecretEnc)
		if err != nil {
			return err
		}
	}
	updates := map[string]any{
		"name": in.Name, "type": in.Type, "endpoint": strings.TrimSpace(in.Endpoint),
		"region": strings.TrimSpace(in.Region), "bucket": strings.TrimSpace(in.Bucket),
		"access_key": strings.TrimSpace(in.AccessKey), "backup_path": in.BackupPath,
		"use_ssl": in.UseSSL, "remark": in.Remark,
	}
	if in.Secret != "" {
		enc, err := s.encryptSecret(in.Secret)
		if err != nil {
			return err
		}
		updates["secret_enc"] = enc
	}
	if err := s.db.Model(row).Updates(updates).Error; err != nil {
		return err
	}
	// 变更后回读再验证（用库里的最终凭据）
	fresh, err := s.ByID(id)
	if err != nil {
		return err
	}
	tgt, err := s.targetForRow(ctx, fresh)
	if err != nil {
		return err
	}
	defer func() { _ = tgt.Close() }()
	if err := tgt.Test(ctx); err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接测试失败: "+err.Error())
	}
	return nil
}

// Delete 删除账号（不动远端数据）。
func (s *StorageService) Delete(id uint) error {
	if err := s.db.Delete(&model.StorageAccount{}, id).Error; err != nil {
		return err
	}
	return nil
}

// targetForRow 按库里账号构造目标（解密凭据）。
func (s *StorageService) targetForRow(ctx context.Context, acct *model.StorageAccount) (storageTarget, error) {
	secret, err := s.decryptSecret(acct.SecretEnc)
	if err != nil {
		return nil, err
	}
	return s.targetFor(acct, secret)
}

// targetFor 按类型构造客户端。
func (s *StorageService) targetFor(acct *model.StorageAccount, secret string) (storageTarget, error) {
	switch acct.Type {
	case "s3":
		return newS3Target(acct, secret)
	case "webdav":
		return newWebdavTarget(acct, secret)
	case "sftp":
		return newSftpTarget(acct, secret)
	}
	return nil, errs.Wrap(errs.ErrBadRequest, "不支持的存储类型: "+acct.Type)
}

// ---- s3 兼容（minio-go 覆盖 S3/MinIO/OSS/COS） ----

func stripScheme(ep string) string {
	ep = strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(ep), "https://"), "http://")
	return strings.TrimSuffix(ep, "/")
}

func newS3Target(acct *model.StorageAccount, secret string) (storageTarget, error) {
	cli, err := minio.New(stripScheme(acct.Endpoint), &minio.Options{
		Creds:  credentials.NewStaticV4(acct.AccessKey, secret, ""),
		Secure: acct.UseSSL,
		Region: acct.Region,
	})
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "s3 参数不合法: "+err.Error())
	}
	return &s3Target{cli: cli, bucket: acct.Bucket, root: acct.BackupPath}, nil
}

type s3Target struct {
	cli    *minio.Client
	bucket string
	root   string
}

func (t *s3Target) Test(ctx context.Context) error {
	ok, err := t.cli.BucketExists(ctx, t.bucket)
	if err != nil {
		return err
	}
	if !ok {
		// 尝试创建（有权限则自愈，无权限报明确错误）
		if cerr := t.cli.MakeBucket(ctx, t.bucket, minio.MakeBucketOptions{}); cerr != nil {
			return fmt.Errorf("bucket %s 不存在且无权创建: %w", t.bucket, cerr)
		}
	}
	return nil
}

func (t *s3Target) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	_, err := t.cli.PutObject(ctx, t.bucket, path.Join(t.root, key), r, size, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

func (t *s3Target) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := t.cli.GetObject(ctx, t.bucket, path.Join(t.root, key), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (t *s3Target) List(ctx context.Context, prefix string) ([]storageObject, error) {
	out := []storageObject{}
	objs := t.cli.ListObjects(ctx, t.bucket, minio.ListObjectsOptions{Prefix: path.Join(t.root, prefix), Recursive: true})
	for o := range objs {
		if o.Err != nil {
			return nil, o.Err
		}
		out = append(out, storageObject{Name: strings.TrimPrefix(o.Key, t.root+"/"), Size: o.Size, ModTime: o.LastModified})
	}
	return out, nil
}

func (t *s3Target) Delete(ctx context.Context, key string) error {
	return t.cli.RemoveObject(ctx, t.bucket, path.Join(t.root, key), minio.RemoveObjectOptions{})
}

func (t *s3Target) Close() error { return nil }

// ---- webdav ----

func newWebdavTarget(acct *model.StorageAccount, secret string) (storageTarget, error) {
	ep := strings.TrimSuffix(strings.TrimSpace(acct.Endpoint), "/")
	cli, err := webdav.NewClient(webdav.HTTPClientWithBasicAuth(&http.Client{Timeout: 60 * time.Second}, acct.AccessKey, secret), ep)
	if err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, "webdav 参数不合法: "+err.Error())
	}
	return &webdavTarget{cli: cli, root: "/" + strings.Trim(acct.BackupPath, "/")}, nil
}

type webdavTarget struct {
	cli  *webdav.Client
	root string
}

func (t *webdavTarget) Test(ctx context.Context) error {
	if _, err := t.cli.Stat(ctx, t.root); err == nil {
		return nil
	}
	if err := t.mkdirAll(ctx, t.root); err != nil {
		return err
	}
	_, err := t.cli.Stat(ctx, t.root)
	return err
}

func (t *webdavTarget) mkdirAll(ctx context.Context, p string) error {
	segs := strings.Split(strings.Trim(p, "/"), "/")
	cur := ""
	for _, seg := range segs {
		if seg == "" {
			continue
		}
		cur += "/" + seg
		if _, err := t.cli.Stat(ctx, cur); err == nil {
			continue
		}
		if err := t.cli.Mkdir(ctx, cur); err != nil {
			// 目录可能已存在（部分服务器 Stat 不支持），Stat 成功即忽略
			if _, serr := t.cli.Stat(ctx, cur); serr == nil {
				continue
			}
			return err
		}
	}
	return nil
}

func (t *webdavTarget) Put(ctx context.Context, key string, r io.Reader, _ int64) error {
	full := path.Join(t.root, key)
	if err := t.mkdirAll(ctx, path.Dir(full)); err != nil {
		return err
	}
	w, err := t.cli.Create(ctx, full)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (t *webdavTarget) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	return t.cli.Open(ctx, path.Join(t.root, key))
}

func (t *webdavTarget) List(ctx context.Context, prefix string) ([]storageObject, error) {
	out := []storageObject{}
	rootPrefix := strings.Trim(path.Join(t.root, prefix), "/") + "/"
	entries, err := t.cli.ReadDir(ctx, t.root, true)
	if err != nil {
		return out, err
	}
	for _, e := range entries {
		if e.IsDir || !strings.HasPrefix(e.Path, rootPrefix) {
			continue
		}
		out = append(out, storageObject{
			Name:    strings.TrimPrefix(e.Path, rootPrefix),
			Size:    e.Size,
			ModTime: e.ModTime,
		})
	}
	return out, nil
}

func (t *webdavTarget) Delete(ctx context.Context, key string) error {
	return t.cli.RemoveAll(ctx, path.Join(t.root, key))
}

func (t *webdavTarget) Close() error { return nil }

// ---- sftp ----

func newSftpTarget(acct *model.StorageAccount, secret string) (storageTarget, error) {
	host := stripScheme(acct.Endpoint)
	if _, _, err := splitHostPort(host, "22"); err != nil {
		return nil, errs.Wrap(errs.ErrBadRequest, err.Error())
	}
	var auth ssh.AuthMethod
	if strings.HasPrefix(strings.TrimSpace(secret), "-----BEGIN") {
		signer, err := ssh.ParsePrivateKey([]byte(secret))
		if err != nil {
			return nil, errs.Wrap(errs.ErrBadRequest, "私钥解析失败: "+err.Error())
		}
		auth = ssh.PublicKeys(signer)
	} else {
		auth = ssh.Password(secret)
	}
	return &sftpTarget{
		host: host, user: acct.AccessKey, auth: auth,
		root: "/" + strings.Trim(acct.BackupPath, "/"),
	}, nil
}

func splitHostPort(host, defPort string) (string, string, error) {
	if i := strings.LastIndex(host, ":"); i >= 0 && !strings.Contains(host[i:], "]") {
		return host[:i], host[i+1:], nil
	}
	return host, defPort, nil
}

type sftpTarget struct {
	host string
	user string
	auth ssh.AuthMethod
	root string

	cli    *sftp.Client
	sshCli *ssh.Client
}

func (t *sftpTarget) connect(ctx context.Context) error {
	if t.cli != nil {
		return nil
	}
	h, port, _ := splitHostPort(t.host, "22")
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", h+":"+port)
	if err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "sftp 连接失败: "+err.Error())
	}
	cfg := &ssh.ClientConfig{
		User:            t.user,
		Auth:            []ssh.AuthMethod{t.auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 内网备份场景；known_hosts 校验后续增强
		Timeout:         15 * time.Second,
	}
	sshCli, chans, reqs, err := ssh.NewClientConn(conn, h+":"+port, cfg)
	if err != nil {
		_ = conn.Close()
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "sftp 握手失败: "+err.Error())
	}
	client := ssh.NewClient(sshCli, chans, reqs)
	cli, err := sftp.NewClient(client)
	if err != nil {
		_ = client.Close()
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "sftp 会话失败: "+err.Error())
	}
	t.cli, t.sshCli = cli, client
	return nil
}

func (t *sftpTarget) Test(ctx context.Context) error {
	if err := t.connect(ctx); err != nil {
		return err
	}
	if _, err := t.cli.Stat(t.root); err != nil {
		if err := t.mkdirAll(t.root); err != nil {
			return err
		}
		if _, err := t.cli.Stat(t.root); err != nil {
			return err
		}
	}
	return nil
}

func (t *sftpTarget) mkdirAll(p string) error {
	segs := strings.Split(strings.Trim(p, "/"), "/")
	cur := ""
	for _, seg := range segs {
		if seg == "" {
			continue
		}
		cur += "/" + seg
		if _, err := t.cli.Stat(cur); err == nil {
			continue
		}
		if err := t.cli.Mkdir(cur); err != nil {
			if _, serr := t.cli.Stat(cur); serr == nil {
				continue
			}
			return err
		}
	}
	return nil
}

func (t *sftpTarget) Put(ctx context.Context, key string, r io.Reader, _ int64) error {
	if err := t.connect(ctx); err != nil {
		return err
	}
	full := path.Join(t.root, key)
	if err := t.mkdirAll(path.Dir(full)); err != nil {
		return err
	}
	w, err := t.cli.Create(full)
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (t *sftpTarget) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := t.connect(ctx); err != nil {
		return nil, err
	}
	return t.cli.Open(path.Join(t.root, key))
}

func (t *sftpTarget) List(ctx context.Context, prefix string) ([]storageObject, error) {
	if err := t.connect(ctx); err != nil {
		return nil, err
	}
	out := []storageObject{}
	walker := t.cli.Walk(path.Join(t.root, prefix))
	for walker.Step() {
		if walker.Err() != nil {
			return nil, walker.Err()
		}
		e := walker.Stat()
		if e == nil || e.IsDir() {
			continue
		}
		full := walker.Path()
		out = append(out, storageObject{
			Name:    strings.TrimPrefix(strings.TrimPrefix(full, t.root), "/"),
			Size:    e.Size(),
			ModTime: e.ModTime(),
		})
	}
	return out, nil
}

func (t *sftpTarget) Delete(ctx context.Context, key string) error {
	if err := t.connect(ctx); err != nil {
		return err
	}
	return t.cli.Remove(path.Join(t.root, key))
}

func (t *sftpTarget) Close() error {
	if t.cli != nil {
		_ = t.cli.Close()
	}
	if t.sshCli != nil {
		return t.sshCli.Close()
	}
	return nil
}

// ---- 面向备份管线的组合操作 ----

// TestConnection 手动连通性测试（返回耗时）。
func (s *StorageService) TestConnection(ctx context.Context, id uint) (map[string]any, error) {
	acct, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	tgt, err := s.targetForRow(ctx, acct)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tgt.Close() }()
	start := time.Now()
	if err := tgt.Test(ctx); err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "连接失败: "+err.Error())
	}
	return map[string]any{"ok": true, "latencyMs": time.Since(start).Milliseconds()}, nil
}

// Objects 远端对象列表（category 为备份类别子目录，如 databases/mysql/acc-mysql）。
func (s *StorageService) Objects(ctx context.Context, id uint, category string) ([]storageObject, error) {
	acct, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	if strings.Contains(category, "..") {
		return nil, errs.ErrPathInvalid
	}
	tgt, err := s.targetForRow(ctx, acct)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tgt.Close() }()
	objs, err := tgt.List(ctx, category)
	if err != nil {
		return nil, err
	}
	for i := range objs {
		objs[i].Size = objs[i].Size / 1024 / 1024
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Name > objs[j].Name })
	return objs, nil
}

// DeleteObject 删除远端对象（key 必须在 BackupPath 下，防误删外部对象）。
func (s *StorageService) DeleteObject(ctx context.Context, id uint, key string) error {
	acct, err := s.ByID(id)
	if err != nil {
		return err
	}
	key = strings.TrimPrefix(path.Clean("/"+key), "/")
	if key == "" || strings.Contains(key, "..") {
		return errs.ErrPathInvalid
	}
	tgt, err := s.targetForRow(ctx, acct)
	if err != nil {
		return err
	}
	defer func() { _ = tgt.Close() }()
	return tgt.Delete(ctx, key)
}

// UploadAgentFile 把 agent 主机侧文件上传到远端（流式），keep>0 时按类别保留份数清理。
// 返回远端 key。上传失败返回错误（本地文件不受影响）。
func (s *StorageService) UploadAgentFile(ctx context.Context, id uint, category, agentPath string, keep int) (string, error) {
	acct, err := s.ByID(id)
	if err != nil {
		return "", err
	}
	if strings.Contains(category, "..") || agentPath == "" || strings.Contains(agentPath, "..") {
		return "", errs.ErrPathInvalid
	}
	name := path.Base(agentPath)
	key := path.Join(strings.Trim(category, "/"), name)
	ac, err := s.client()
	if err != nil {
		return "", err
	}
	tgt, err := s.targetForRow(ctx, acct)
	if err != nil {
		return "", err
	}
	defer func() { _ = tgt.Close() }()
	// agent 下载流（Authorization 由 agentclient 注入）
	req, err := ac.NewRequest(ctx, http.MethodGet, "/agent/v1/files/download?path="+url.QueryEscape(agentPath), nil)
	if err != nil {
		return "", err
	}
	resp, err := ac.HTTP.Do(req)
	if err != nil {
		return "", errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "读取本地备份失败: "+err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return "", errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", fmt.Sprintf("读取本地备份失败: HTTP %d", resp.StatusCode))
	}
	perr := tgt.Put(ctx, key, resp.Body, -1)
	_ = resp.Body.Close()
	if perr != nil {
		return "", errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "上传远端失败: "+perr.Error())
	}
	if keep > 0 {
		if rerr := s.applyRetention(ctx, tgt, category, keep); rerr != nil {
			// 清理失败不影响本次上传结果，仅回传提示
			return key, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "已上传，但保留策略清理失败: "+rerr.Error())
		}
	}
	return key, nil
}

// applyRetention 类别前缀下按名字倒序保留 keep 份（备份名内嵌时间戳，字典序=时间序）。
func (s *StorageService) applyRetention(ctx context.Context, tgt storageTarget, category string, keep int) error {
	prefix := strings.Trim(category, "/")
	if prefix == "" {
		return nil
	}
	objs, err := tgt.List(ctx, prefix+"/")
	if err != nil {
		return err
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].Name > objs[j].Name })
	var lastErr error
	for i := keep; i < len(objs); i++ {
		if derr := tgt.Delete(ctx, objs[i].Name); derr != nil {
			lastErr = derr
		}
	}
	return lastErr
}

// Fetch 把远端对象拉回 agent 本地目录（流式 multipart 转发，供恢复/导入流程使用）。
func (s *StorageService) Fetch(ctx context.Context, id uint, key, destDir string) (map[string]any, error) {
	acct, err := s.ByID(id)
	if err != nil {
		return nil, err
	}
	key = strings.TrimPrefix(path.Clean("/"+key), "/")
	if key == "" || strings.Contains(key, "..") || destDir == "" || strings.Contains(destDir, "..") {
		return nil, errs.ErrPathInvalid
	}
	name := path.Base(key)
	tgt, err := s.targetForRow(ctx, acct)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tgt.Close() }()
	rc, err := tgt.Open(ctx, key)
	if err != nil {
		return nil, errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", "读取远端对象失败: "+err.Error())
	}
	defer func() { _ = rc.Close() }()
	ac, err := s.client()
	if err != nil {
		return nil, err
	}
	if err := UploadMultipartToAgent(ac, ctx, destDir, name, rc); err != nil {
		return nil, err
	}
	return map[string]any{"file": name, "path": path.Join(destDir, name)}, nil
}

// UploadMultipartToAgent 流式构造 multipart 转发到 agent files/upload（与 api 层 uploadMultipart 同契约）。
func UploadMultipartToAgent(ac *agentclient.Client, ctx context.Context, dir, filename string, r io.Reader) error {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		var werr error
		if part, perr := mw.CreateFormFile("file", filename); perr == nil {
			_, werr = io.Copy(part, r)
		} else {
			werr = perr
		}
		_ = mw.Close()
		_ = pw.CloseWithError(werr)
	}()
	req, err := ac.NewRequest(ctx, http.MethodPost, "/agent/v1/files/upload?path="+url.QueryEscape(dir), pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := ac.HTTP.Do(req)
	if err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return errs.New(errs.CodeAgentUnreach, "error.agentUnreachable", err.Error())
	}
	if env.Code != 0 {
		return &errs.Error{Code: env.Code, Message: env.Message}
	}
	return nil
}
