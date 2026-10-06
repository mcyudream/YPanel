package dto

import "time"

// FileEntry 文件/目录条目。
type FileEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"` // 绝对路径
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	Mode    string    `json:"mode"`    // 权限串，如 drwxr-xr-x
	ModeOct string    `json:"modeOct"` // 八进制，如 0755
	Owner   string    `json:"owner"`
	Group   string    `json:"group"`
	ModTime time.Time `json:"modTime"`
	Target  string    `json:"target,omitempty"` // 符号链接目标
}

// FileListResp 目录列表。
type FileListResp struct {
	Path    string      `json:"path"`
	Entries []FileEntry `json:"entries"`
}

// FileWriteReq 写文本文件。
type FileWriteReq struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content"`
}

// FileMkdirReq 创建目录（递归）。
type FileMkdirReq struct {
	Path string `json:"path" binding:"required"`
}

// FileRenameReq 重命名/移动。
type FileRenameReq struct {
	From string `json:"from" binding:"required"`
	To   string `json:"to" binding:"required"`
}

// FileDeleteReq 删除（文件/目录，递归）。
type FileDeleteReq struct {
	Paths []string `json:"paths" binding:"required,min=1"`
}

// FileReadResp 读文本文件（限长）。
type FileReadResp struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Size      int64  `json:"size"`
	Truncated bool   `json:"truncated"` // 超出读取上限被截断
}

// FileChmodReq 权限修改请求。
type FileChmodReq struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode" binding:"required"`
}

// FileCompressReq 压缩请求（tar.gz）。
type FileCompressReq struct {
	Src  string `json:"src" binding:"required"`
	Dest string `json:"dest" binding:"required"`
}

// FileDecompressReq 解压请求。
type FileDecompressReq struct {
	Archive string `json:"archive" binding:"required"`
	DestDir string `json:"destDir" binding:"required"`
}
