// Package service 业务层：认证、用户、设置、节点（agent 生命周期）。
package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// randomHex 生成 n 字节随机数的十六进制串。
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // 系统熵源不可用属启动期致命错误
	}
	return hex.EncodeToString(b)
}

// SettingService 键值设置（带进程内缓存）。
type SettingService struct {
	db  *gorm.DB
	mem map[string]string
}

// NewSettingService 创建设置服务。
func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{db: db, mem: map[string]string{}}
}

// Get 读取设置，不存在返回默认值。
func (s *SettingService) Get(key, def string) string {
	if v, ok := s.mem[key]; ok {
		return v
	}
	var row model.Setting
	if err := s.db.Where("`key` = ?", key).First(&row).Error; err != nil {
		return def
	}
	s.mem[key] = row.Value
	return row.Value
}

// Set 写设置（upsert）。
func (s *SettingService) Set(key, value string) error {
	if err := s.db.Where("`key` = ?", key).Assign(model.Setting{Value: value}).FirstOrCreate(&model.Setting{Key: key}).Error; err != nil {
		return err
	}
	s.mem[key] = value
	return nil
}

// GetOrCreate 首启生成一次并固化的值（如 jwt 密钥）。
func (s *SettingService) GetOrCreate(key, genValue string) (string, error) {
	if v := s.Get(key, ""); v != "" {
		return v, nil
	}
	if err := s.Set(key, genValue); err != nil {
		return "", err
	}
	return genValue, nil
}

// UserService 用户管理。
type UserService struct {
	db *gorm.DB
}

// NewUserService 创建用户服务。
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// EnsureBootstrap 首启建 admin：密码取 cfg；否则随机生成并写入 <data>/admin-passwd.txt。
func (s *UserService) EnsureBootstrap(dataDir, configured string) (generated string, err error) {
	var count int64
	if err := s.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}
	pwd := configured
	if pwd == "" {
		pwd = randomHex(8)
		generated = pwd
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	admin := &model.User{Username: "admin", Password: string(hash), Nickname: "管理员", Role: "admin", Status: 1, TokenVersion: 1}
	if err := s.db.Create(admin).Error; err != nil {
		return "", err
	}
	slog.Info("已创建初始管理员 admin")
	if generated != "" {
		p := filepath.Join(dataDir, "admin-passwd.txt")
		if err := os.WriteFile(p, []byte("admin / "+generated+"\n（首次登录后请立即修改密码，本文件可删除）\n"), 0o600); err != nil {
			return generated, fmt.Errorf("初始密码落盘失败: %w", err)
		}
		slog.Warn("随机初始密码已写入", "file", p)
	}
	return generated, nil
}

// ResetPassword 重置用户密码（CLI 通道）。
func (s *UserService) ResetPassword(username, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res := s.db.Model(&model.User{}).Where("username = ?", username).
		Updates(map[string]any{"password": string(hash), "token_version": gorm.Expr("token_version + 1")})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.New(errs.CodeNotFound, "error.userNotFound", "用户不存在")
	}
	return nil
}

// Auth 认证与会话。
type Auth struct {
	db       *gorm.DB
	settings *SettingService
	secret   []byte
	fails    map[string]*failRecord // username|ip → 失败计数
}

type failRecord struct {
	count int
	last  time.Time
	lock  time.Time
}

// NewAuth 创建认证服务并装载 JWT 密钥。
func NewAuth(db *gorm.DB, settings *SettingService) (*Auth, error) {
	secret, err := settings.GetOrCreate("jwt_secret", randomHex(32))
	if err != nil {
		return nil, err
	}
	return &Auth{db: db, settings: settings, secret: []byte(secret), fails: map[string]*failRecord{}}, nil
}

const (
	maxFails     = 5
	failWindow   = 10 * time.Minute
	lockDuration = 15 * time.Minute
)

// Login 登录校验：锁定检查 → 用户校验 → 审计落库。
func (a *Auth) Login(username, password, ip, ua string) (*model.User, error) {
	key := username + "|" + ip
	if fr, ok := a.fails[key]; ok && fr.lock.After(time.Now()) {
		a.writeLog(username, ip, ua, false, "尝试过于频繁，已临时锁定")
		return nil, errs.New(errs.CodeForbidden, "error.loginLocked", "失败次数过多，请 15 分钟后重试")
	}

	user, err := a.byUsername(username)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		a.recordFail(key)
		msg := "用户名或密码错误"
		if err == gorm.ErrRecordNotFound {
			msg = "用户不存在"
		}
		a.writeLog(username, ip, ua, false, msg)
		return nil, errs.New(errs.CodeUserOrPassWrong, "error.loginFailed", "用户名或密码错误")
	}
	if user.Status != 1 {
		a.writeLog(username, ip, ua, false, "账号已禁用")
		return nil, errs.New(errs.CodeUserDisabled, "error.userDisabled", "账号已禁用")
	}
	delete(a.fails, key)
	now := time.Now()
	_ = a.db.Model(user).Updates(map[string]any{"last_login_at": now}).Error
	a.writeLog(username, ip, ua, true, "登录成功")
	return user, nil
}

func (a *Auth) recordFail(key string) {
	fr, ok := a.fails[key]
	if !ok || time.Since(fr.last) > failWindow {
		fr = &failRecord{}
		a.fails[key] = fr
	}
	fr.count++
	fr.last = time.Now()
	if fr.count >= maxFails {
		fr.lock = time.Now().Add(lockDuration)
	}
}

func (a *Auth) writeLog(username, ip, ua string, success bool, msg string) {
	_ = a.db.Create(&model.LoginLog{Username: username, IP: ip, UserAgent: ua, Success: success, Message: msg}).Error
}

func (a *Auth) byUsername(username string) (*model.User, error) {
	var u model.User
	if err := a.db.Where("username = ?", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ByID 按 ID 查用户。
func (a *Auth) ByID(id uint) (*model.User, error) {
	var u model.User
	if err := a.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ChangePassword 修改自身密码并使全部旧会话失效。
func (a *Auth) ChangePassword(id uint, oldPwd, newPwd string) error {
	u, err := a.ByID(id)
	if err != nil {
		return errs.ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(oldPwd)) != nil {
		return errs.New(errs.CodeOldPassWrong, "error.oldPassWrong", "原密码错误")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return a.db.Model(u).Updates(map[string]any{"password": string(hash), "token_version": u.TokenVersion + 1}).Error
}

// Secret JWT 签名密钥。
func (a *Auth) Secret() []byte { return a.secret }

// DB 供审计查询。
func (a *Auth) DB() *gorm.DB { return a.db }
