package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ypanel/core/internal/middleware"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/service"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

func userInfoOf(u *model.User) dto.UserInfo {
	return dto.UserInfo{
		ID:          u.ID,
		Username:    u.Username,
		Nickname:    u.Nickname,
		Role:        u.Role,
		LastLoginAt: u.LastLoginAt,
	}
}

// AuthAPI 认证接口。
type AuthAPI struct {
	Auth    *service.Auth
	Sec     *service.SecuritySettingsService
	Version string
}

// TwoFASetup POST /api/v1/auth/2fa/setup（admin：生成密钥，返回 otpauth URI）
func (a *AuthAPI) TwoFASetup(c *gin.Context) {
	secret, uri, err := a.Sec.Enable2FA(c.Request.Context(), "YPanel", "admin")
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"secret": secret, "otpauthUri": uri})
}

// TwoFADisable POST /api/v1/auth/2fa/disable（admin）
func (a *AuthAPI) TwoFADisable(c *gin.Context) {
	if err := a.Sec.Disable2FA(c.Request.Context()); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// TwoFAStatus GET /api/v1/auth/2fa/status
func (a *AuthAPI) TwoFAStatus(c *gin.Context) {
	respOK(c, gin.H{"enabled": a.Sec.TwoFAEnabled()})
}

// Login POST /api/v1/auth/login
func (a *AuthAPI) Login(c *gin.Context) {
	req, ok := bind[dto.LoginReq](c)
	if !ok {
		return
	}
	user, err := a.Auth.Login(req.Username, req.Password, req.OtpCode, clientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		respErr(c, err)
		return
	}
	token, exp, err := a.Auth.IssueToken(user)
	if err != nil {
		respErr(c, err)
		return
	}

	// M49：会话 cookie——安全入口门禁凭它放行已登录浏览器的根路径刷新/直达（HttpOnly，30 天）
	c.SetCookie("yp_entry_ok", "1", 30*24*3600, "/", "", false, true)
	respOK(c, dto.LoginResp{Token: token, ExpireAt: exp, User: userInfoOf(user)})
}

// Me GET /api/v1/auth/me
func (a *AuthAPI) Me(c *gin.Context) {
	user, err := a.Auth.ByID(c.GetUint(middleware.CtxUID))
	if err != nil {
		respErr(c, errs.ErrUnauthorized)
		return
	}
	respOK(c, userInfoOf(user))
}

// Logout POST /api/v1/auth/logout（无状态 JWT：服务端无会话可销毁，前端丢弃 token）
func (a *AuthAPI) Logout(c *gin.Context) {
	// M49：同步清除入口会话 cookie——登出后该浏览器回到「必须走入口」状态
	c.SetCookie("yp_entry_ok", "", -1, "/", "", false, true)
	respOK(c, struct{}{})
}

// ChangePassword PUT /api/v1/auth/password
func (a *AuthAPI) ChangePassword(c *gin.Context) {
	req, ok := bind[dto.ChangePasswordReq](c)
	if !ok {
		return
	}
	if err := a.Auth.ChangePassword(c.GetUint(middleware.CtxUID), req.OldPassword, req.NewPassword); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// UserAPI 用户管理接口（admin）。
type UserAPI struct {
	DB *gorm.DB
}

// List GET /api/v1/users
func (u *UserAPI) List(c *gin.Context) {
	page, size := pageParams(c)
	var total int64
	var rows []model.User
	q := u.DB.Model(&model.User{})
	if err := q.Count(&total).Error; err != nil {
		respErr(c, err)
		return
	}
	if err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		respErr(c, err)
		return
	}
	items := make([]dto.UserInfo, 0, len(rows))
	for i := range rows {
		items = append(items, userInfoOf(&rows[i]))
	}
	respOK(c, dto.NewPageResp(total, items))
}

// Create POST /api/v1/users
func (u *UserAPI) Create(c *gin.Context) {
	req, ok := bind[dto.UserCreateReq](c)
	if !ok {
		return
	}
	var count int64
	_ = u.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error
	if count > 0 {
		respErr(c, errs.New(errs.CodeConflict, "error.userExists", "用户名已存在"))
		return
	}
	user, err := u.create(req.Username, req.Password, req.Nickname, req.Role)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, userInfoOf(user))
}

// Update PUT /api/v1/users/:id
func (u *UserAPI) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	req, ok := bind[dto.UserUpdateReq](c)
	if !ok {
		return
	}
	var user model.User
	if err := u.DB.First(&user, id).Error; err != nil {
		respErr(c, errs.New(errs.CodeNotFound, "error.userNotFound", "用户不存在"))
		return
	}
	updates := map[string]any{}
	if req.Nickname != nil {
		updates["nickname"] = *req.Nickname
	}
	if req.Role != nil {
		updates["role"] = *req.Role
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Password != nil {
		hash, err := u.hashPassword(*req.Password)
		if err != nil {
			respErr(c, err)
			return
		}
		updates["password"] = hash
		updates["token_version"] = user.TokenVersion + 1 // 改密强制下线
	}
	if len(updates) > 0 {
		if err := u.DB.Model(&user).Updates(updates).Error; err != nil {
			respErr(c, err)
			return
		}
	}
	respOK(c, userInfoOf(&user))
}

// Delete DELETE /api/v1/users/:id
func (u *UserAPI) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respErr(c, errs.ErrBadRequest)
		return
	}
	if uint(id) == c.GetUint(middleware.CtxUID) {
		respErr(c, errs.New(errs.CodeBadRequest, "error.cannotDeleteSelf", "不能删除自己"))
		return
	}
	if err := u.DB.Delete(&model.User{}, id).Error; err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// LoginLogs GET /api/v1/audit/logins
func (u *UserAPI) LoginLogs(c *gin.Context) {
	page, size := pageParams(c)
	var total int64
	var rows []model.LoginLog
	q := u.DB.Model(&model.LoginLog{})
	if err := q.Count(&total).Error; err != nil {
		respErr(c, err)
		return
	}
	if err := q.Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		respErr(c, err)
		return
	}
	items := make([]dto.LoginLogItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, dto.LoginLogItem{
			ID: r.ID, Username: r.Username, IP: r.IP, UserAgent: r.UserAgent,
			Success: r.Success, Message: r.Message, CreatedAt: r.CreatedAt,
		})
	}
	respOK(c, dto.NewPageResp(total, items))
}

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 200 {
		size = 20
	}
	return page, size
}
