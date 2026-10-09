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
	Path     string `json:"path" binding:"required"`
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty"` // 落盘编码，默认 utf-8（白名单见 agent files 包）
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

// FileCopyReq 复制（文件/目录，递归；目标已存在默认报错，overwrite=true 时先移除目标再复制）。
type FileCopyReq struct {
	From      string `json:"from" binding:"required"`
	To        string `json:"to" binding:"required"`
	Overwrite bool   `json:"overwrite,omitempty"`
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
	// 以下字段仅 raw=1 读取时返回（编辑器编码检测用）
	ContentB64 string `json:"contentB64,omitempty"` // 原始字节 base64
	Encoding   string `json:"encoding,omitempty"`   // 请求中实际应用的服务端编码
	IsBinary   bool   `json:"isBinary,omitempty"`   // 检测到 NUL 字节，视为二进制
}

// FileChmodReq 权限修改请求。
type FileChmodReq struct {
	Path      string `json:"path" binding:"required"`
	Mode      string `json:"mode" binding:"required"`
	Recursive bool   `json:"recursive,omitempty"` // 应用到子文件/子目录
}

// FileChownReq 属主修改请求（owner/group 传名字或数字 ID，均为空 = 不修改；空串表示该项不更改）。
type FileChownReq struct {
	Path      string `json:"path" binding:"required"`
	Owner     string `json:"owner,omitempty"`
	Group     string `json:"group,omitempty"`
	Recursive bool   `json:"recursive,omitempty"`
}

// FileOwnerEntry 用户/组条目。
type FileOwnerEntry struct {
	Name string `json:"name"`
	ID   string `json:"id"` // uid / gid 数字串
}

// FileOwnersResp 系统用户与组列表（来自 /etc/passwd、/etc/group）。
type FileOwnersResp struct {
	Users  []FileOwnerEntry `json:"users"`
	Groups []FileOwnerEntry `json:"groups"`
}

// FileCompressReq 压缩请求（tar.gz，支持多源打包）。
type FileCompressReq struct {
	Src  string   `json:"src,omitempty"`  // 单源（兼容旧调用方）
	Srcs []string `json:"srcs,omitempty"` // 多源
	Dest string   `json:"dest" binding:"required"`
}

// FileDecompressReq 解压请求。
type FileDecompressReq struct {
	Archive string `json:"archive" binding:"required"`
	DestDir string `json:"destDir" binding:"required"`
}

// TrashItemMeta 回收站条目（M38）。
type TrashItemMeta struct {
	Original  string    `json:"original"`
	Name      string    `json:"name"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
	TrashedAt time.Time `json:"trashedAt"`
}

// FileTrashReq 移入回收站。
type FileTrashReq struct {
	Paths []string `json:"paths" binding:"required,min=1"`
}

// FileTrashNamesReq 回收站按名操作（还原/彻底删除）。
type FileTrashNamesReq struct {
	Names []string `json:"names" binding:"required,min=1"`
}

// FileTrashListResp 回收站列表。
type FileTrashListResp struct {
	Items []TrashItemMeta `json:"items"`
}

// FileRemoteDownloadReq 远程 URL 下载（M38，core 侧 SSRF 校验后经 agent 落盘）。
type FileRemoteDownloadReq struct {
	URL     string `json:"url" binding:"required"`
	DestDir string `json:"destDir" binding:"required"`
}
