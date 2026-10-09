// 磁盘占用分析 handlers。
package server

import (
	"net/http"

	"github.com/ypanel/agent/internal/diskusage"
)

// handleDiskUsageTree GET /agent/v1/disk/usage?path=/var[&refresh=1]（du 语义目录占用；24h 缓存，refresh=1 绕过）
func (s *Server) handleDiskUsageTree(w http.ResponseWriter, r *http.Request) {
	out, err := diskusage.Tree(r.Context(), r.URL.Query().Get("path"), r.URL.Query().Get("refresh") == "1")
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, out)
}
