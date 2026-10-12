// DB Admin S3 对象工作台服务（M68）：bucket≈库、对象≈行。
// 列表/删除走本服务；上传/下载为二进制流，API 层直接经 S3Browser 通道处理，写路径全审计。
package service

import (
	"context"
	"io"

	"github.com/ypanel/core/internal/dbdriver"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/shared/errs"
)

// S3Objects 对象分页列表（前缀即目录）。
func (s *DBAdminService) S3Objects(ctx context.Context, instanceID uint, bucket, prefix, startAfter string, limit int) (*dbdriver.S3ObjectPage, error) {
	var out *dbdriver.S3ObjectPage
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, _ *model.DatabaseInstance, drv dbdriver.Driver) error {
		sb, ok := drv.(dbdriver.S3Browser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 S3 兼容对象存储")
		}
		var err error
		out, err = sb.ListObjects(ctx, bucket, prefix, startAfter, limit)
		return err
	})
	return out, err
}

// S3ObjectDelete 删除对象（审计）。
func (s *DBAdminService) S3ObjectDelete(ctx context.Context, instanceID uint, username, bucket, key string) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		sb, ok := drv.(dbdriver.S3Browser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 S3 兼容对象存储")
		}
		err := sb.DeleteObject(ctx, bucket, key)
		s.audit(username, inst, "s3_object_delete", bucket, "key="+key, err == nil)
		return err
	})
}

// S3ObjectUpload 上传对象（审计；r 由 API 层提供，size 可为 -1 交给 SDK 分片）。
func (s *DBAdminService) S3ObjectUpload(ctx context.Context, instanceID uint, username, bucket, key string, r io.Reader, size int64) error {
	return s.withDriver(ctx, instanceID, dbAdminWriteTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		sb, ok := drv.(dbdriver.S3Browser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 S3 兼容对象存储")
		}
		err := sb.PutObject(ctx, bucket, key, r, size)
		s.audit(username, inst, "s3_object_upload", bucket, "key="+key, err == nil)
		return err
	})
}

// S3ObjectRead 打开对象读流（下载用；调用方负责 Close；审计）。
func (s *DBAdminService) S3ObjectRead(ctx context.Context, instanceID uint, username, bucket, key string) (io.ReadCloser, int64, error) {
	var rc io.ReadCloser
	var size int64
	err := s.withDriver(ctx, instanceID, dbAdminQueryTimeout, func(ctx context.Context, inst *model.DatabaseInstance, drv dbdriver.Driver) error {
		sb, ok := drv.(dbdriver.S3Browser)
		if !ok {
			return errs.Wrap(errs.ErrBadRequest, "该实例不是 S3 兼容对象存储")
		}
		var err error
		rc, size, err = sb.GetObject(ctx, bucket, key)
		s.audit(username, inst, "s3_object_download", bucket, "key="+key, err == nil)
		return err
	})
	if err != nil {
		return nil, 0, err
	}
	return rc, size, nil
}
