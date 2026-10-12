// S3 兼容对象存储驱动（M68）：RustFS/MinIO 等经 S3 API 纳管。
// bucket≈库（Database/ListDatabases 语义化）、对象≈行（走 S3Browser 专用通道，不进通用 RowEdit）。
// 用户/授权管理对象存储无对应概念，明确拒绝。
package dbdriver

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/ypanel/shared/errs"
)

type s3Driver struct {
	client *minio.Client
}

func newS3(host string, port int, accessKey, secretKey string, secure bool) (Driver, error) {
	client, err := minio.New(fmt.Sprintf("%s:%d", host, port), &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, errs.Wrapc(errs.CodeFileOpFailed, err.Error())
	}
	return &s3Driver{client: client}, nil
}

func (d *s3Driver) Close() {}

func (d *s3Driver) Ping(ctx context.Context) error {
	// ListBuckets 兼容只读 AK 的最小探测
	_, err := d.client.ListBuckets(ctx)
	return err
}

// ListDatabases 桶清单（size 不取——全量统计代价大，列表语义优先）。
func (d *s3Driver) ListDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	buckets, err := d.client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DatabaseInfo, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, DatabaseInfo{Name: b.Name})
	}
	return out, nil
}

// CreateDatabase 建桶。charset 参数复用为 region（空取 us-east-1 语义，由服务端决定）。
func (d *s3Driver) CreateDatabase(ctx context.Context, name, charset string) error {
	if err := ValidateBucketName(name); err != nil {
		return err
	}
	return d.client.MakeBucket(ctx, name, minio.MakeBucketOptions{Region: charset})
}

// DropDatabase 删桶（要求桶为空；非空报错提示先清空，防误删批量数据）。
func (d *s3Driver) DropDatabase(ctx context.Context, name string) error {
	if err := ValidateBucketName(name); err != nil {
		return err
	}
	return d.client.RemoveBucket(ctx, name)
}

func (d *s3Driver) ListUsers(ctx context.Context) ([]UserInfo, error) {
	return []UserInfo{}, nil // AK/SK 即实例凭据，无子账号语义（RustFS/MinIO IAM 不纳管）
}

func (d *s3Driver) CreateUser(ctx context.Context, name, host, password string) error {
	return errs.Wrapc(errs.CodeBadRequest, "对象存储无子账号语义，凭据在实例级管理")
}

func (d *s3Driver) DropUser(ctx context.Context, name, host string) error {
	return errs.Wrapc(errs.CodeBadRequest, "对象存储无子账号语义，凭据在实例级管理")
}

func (d *s3Driver) ChangePassword(ctx context.Context, name, host, password string) error {
	return errs.Wrapc(errs.CodeBadRequest, "对象存储无子账号语义，凭据在实例级管理")
}

func (d *s3Driver) EnableRemote(ctx context.Context, password string) error {
	return errs.Wrapc(errs.CodeBadRequest, "对象存储无 SQL 远程授权语义")
}

func (d *s3Driver) DisableRemote(ctx context.Context) error {
	return errs.Wrapc(errs.CodeBadRequest, "对象存储无 SQL 远程授权语义")
}

// ---- S3Browser 能力 ----

func (d *s3Driver) ListObjects(ctx context.Context, bucket, prefix, startAfter string, limit int) (*S3ObjectPage, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	opts := minio.ListObjectsOptions{
		Prefix:     prefix,
		Recursive:  false,
		StartAfter: startAfter,
		MaxKeys:    limit + 1,
	}
	page := &S3ObjectPage{Prefix: prefix, Objects: make([]S3Object, 0, limit)}
	seenDir := map[string]bool{}
	for obj := range d.client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		// 非递归模式下「目录」以 CommonPrefixes 聚合（minio-go 折叠成 key 以 / 结尾的空对象）
		if strings.HasSuffix(obj.Key, "/") {
			if obj.Size == 0 && !seenDir[obj.Key] {
				seenDir[obj.Key] = true
				page.Objects = append(page.Objects, S3Object{Key: obj.Key, IsDir: true, LastModified: obj.LastModified.Format(time.RFC3339)})
			}
		} else {
			page.Objects = append(page.Objects, S3Object{Key: obj.Key, Size: obj.Size, LastModified: obj.LastModified.Format(time.RFC3339)})
		}
		if len(page.Objects) >= limit {
			page.Truncated = true
			break
		}
	}
	return page, nil
}

func (d *s3Driver) DeleteObject(ctx context.Context, bucket, key string) error {
	if strings.TrimSpace(key) == "" {
		return errs.Wrapc(errs.CodeBadRequest, "对象 key 为空")
	}
	return d.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

func (d *s3Driver) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64) error {
	if strings.TrimSpace(key) == "" {
		return errs.Wrapc(errs.CodeBadRequest, "对象 key 为空")
	}
	_, err := d.client.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{})
	return err
}

func (d *s3Driver) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, int64, error) {
	if strings.TrimSpace(key) == "" {
		return nil, 0, errs.Wrapc(errs.CodeBadRequest, "对象 key 为空")
	}
	obj, err := d.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, 0, err
	}
	stat, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, 0, err
	}
	return obj, stat.Size, nil
}

// ValidateBucketName 桶名白名单（S3 规约：3-63 位小写字母/数字/中划线/点）。
func ValidateBucketName(s string) error {
	if len(s) < 3 || len(s) > 63 {
		return errs.Wrap(errs.ErrBadRequest, "桶名长度需 3-63 位: "+s)
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '.') {
			return errs.Wrap(errs.ErrBadRequest, "桶名仅允许小写字母/数字/中划线/点: "+s)
		}
	}
	return nil
}
