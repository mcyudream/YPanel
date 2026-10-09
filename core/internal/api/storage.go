// StorageAPI 远程备份存储接口（M34）。凭据只进不出；全部挂 admin 组。
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/ypanel/core/internal/service"
)

// StorageAPI 存储账号 + 目录/编排备份。
type StorageAPI struct {
	Svc *service.StorageService
	BK  *service.BackupService
}

// List GET /api/v1/storage-accounts
func (a *StorageAPI) List(c *gin.Context) {
	out, err := a.Svc.List()
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Create POST /api/v1/storage-accounts
func (a *StorageAPI) Create(c *gin.Context) {
	req, ok := bind[struct {
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type" binding:"required,oneof=s3 webdav sftp"`
		Endpoint   string `json:"endpoint" binding:"required"`
		Region     string `json:"region"`
		Bucket     string `json:"bucket"`
		AccessKey  string `json:"accessKey"`
		Secret     string `json:"secret" binding:"required"`
		BackupPath string `json:"backupPath"`
		UseSSL     *bool  `json:"useSSL"`
		Remark     string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	in := service.StorageAccountInput{
		Name: req.Name, Type: req.Type, Endpoint: req.Endpoint, Region: req.Region,
		Bucket: req.Bucket, AccessKey: req.AccessKey, Secret: req.Secret,
		BackupPath: req.BackupPath, UseSSL: req.UseSSL == nil || *req.UseSSL, Remark: req.Remark,
	}
	out, err := a.Svc.Create(c.Request.Context(), in)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, gin.H{"id": out.ID, "name": out.Name})
}

// Update PUT /api/v1/storage-accounts/:id（secret 留空保留原值）
func (a *StorageAPI) Update(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Name       string `json:"name" binding:"required"`
		Type       string `json:"type" binding:"required,oneof=s3 webdav sftp"`
		Endpoint   string `json:"endpoint" binding:"required"`
		Region     string `json:"region"`
		Bucket     string `json:"bucket"`
		AccessKey  string `json:"accessKey"`
		Secret     string `json:"secret"`
		BackupPath string `json:"backupPath"`
		UseSSL     *bool  `json:"useSSL"`
		Remark     string `json:"remark"`
	}](c)
	if !ok {
		return
	}
	in := service.StorageAccountInput{
		Name: req.Name, Type: req.Type, Endpoint: req.Endpoint, Region: req.Region,
		Bucket: req.Bucket, AccessKey: req.AccessKey, Secret: req.Secret,
		BackupPath: req.BackupPath, UseSSL: req.UseSSL == nil || *req.UseSSL, Remark: req.Remark,
	}
	if err := a.Svc.Update(c.Request.Context(), id, in); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Delete DELETE /api/v1/storage-accounts/:id
func (a *StorageAPI) Delete(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Svc.Delete(id); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Test POST /api/v1/storage-accounts/:id/test
func (a *StorageAPI) Test(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Svc.TestConnection(c.Request.Context(), id)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// Objects GET /api/v1/storage-accounts/:id/objects?category=
func (a *StorageAPI) Objects(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	out, err := a.Svc.Objects(c.Request.Context(), id, c.Query("category"))
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DeleteObject DELETE /api/v1/storage-accounts/:id/object?key=
func (a *StorageAPI) DeleteObject(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	if err := a.Svc.DeleteObject(c.Request.Context(), id, c.Query("key")); err != nil {
		respErr(c, err)
		return
	}
	respOK(c, struct{}{})
}

// Fetch POST /api/v1/storage-accounts/:id/fetch {key, destDir}（远端对象拉回本地目录）
func (a *StorageAPI) Fetch(c *gin.Context) {
	id, err := idParam(c)
	if err != nil {
		respErr(c, err)
		return
	}
	req, ok := bind[struct {
		Key     string `json:"key" binding:"required"`
		DestDir string `json:"destDir" binding:"required"`
	}](c)
	if !ok {
		return
	}
	out, err := a.Svc.Fetch(c.Request.Context(), id, req.Key, req.DestDir)
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// DirBackup POST /api/v1/backups/dir {srcDir,name,storageAccountId,keep}
func (a *StorageAPI) DirBackup(c *gin.Context) {
	req, ok := bind[struct {
		SrcDir           string `json:"srcDir" binding:"required"`
		Name             string `json:"name"`
		StorageAccountId uint   `json:"storageAccountId"`
		Keep             int    `json:"keep"`
	}](c)
	if !ok {
		return
	}
	out, err := a.BK.DirBackup(c.Request.Context(), req.SrcDir, req.Name,
		service.BackupUploadOpts{StorageAccountID: req.StorageAccountId, Keep: req.Keep})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}

// ComposeBackup POST /api/v1/backups/compose {project,storageAccountId,keep}
func (a *StorageAPI) ComposeBackup(c *gin.Context) {
	req, ok := bind[struct {
		Project          string `json:"project" binding:"required"`
		StorageAccountId uint   `json:"storageAccountId"`
		Keep             int    `json:"keep"`
	}](c)
	if !ok {
		return
	}
	out, err := a.BK.ComposeBackup(c.Request.Context(), req.Project,
		service.BackupUploadOpts{StorageAccountID: req.StorageAccountId, Keep: req.Keep})
	if err != nil {
		respErr(c, err)
		return
	}
	respOK(c, out)
}
