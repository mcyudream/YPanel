package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// settingsWhitelist 可经 API 修改的设置键（白名单）。
var settingsWhitelist = map[string]bool{
	"panel.name":       true, // 面板名称
	"panel.language":   true, // 语言 zh-CN / en
	"panel.theme.mode": true, // 默认主题 light / dark / auto
	"panel.entry":      true, // 安全入口路径（空 = 关闭）
}

// SettingsAPI 面板设置。
type SettingsAPI struct {
	Settings *service.SettingService
}

// Get GET /api/v1/settings → 全量白名单键值
func (s *SettingsAPI) Get(c *gin.Context) {
	out := map[string]string{}
	for k := range settingsWhitelist {
		out[k] = s.Settings.Get(k, "")
	}
	respOK(c, out)
}

// Put PUT /api/v1/settings {key: value, ...}
func (s *SettingsAPI) Put(c *gin.Context) {
	req, ok := bind[map[string]string](c)
	if !ok {
		return
	}
	for k := range *req {
		if !settingsWhitelist[k] {
			respErr(c, errBadRequest("设置项不在白名单: "+k))
			return
		}
	}
	for k, v := range *req {
		if err := s.Settings.Set(k, v); err != nil {
			respErr(c, err)
			return
		}
	}
	s.Get(c)
}
