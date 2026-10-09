// 节点命令执行与文件原子写入的共享工具（M27 内网 DNS / M28 hosts 复用）。
// 执行通道：core → agent /agent/v1/exec（sh -c 语义），本机（内嵌 agent）与远程节点同构。
package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

// execOnNode 在目标节点执行命令，返回 (output, exitCode)。
func execOnNode(ctx context.Context, nodes *NodeService, nodeId, cmd string, timeoutSecs int) (string, int, error) {
	node, err := nodes.ByID(nodeId)
	if err != nil {
		return "", 0, err
	}
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](agentclient.New(node.BaseURL, node.Token), ctx, "POST", "/agent/v1/exec",
		&dto.ExecReq{Command: cmd, TimeoutSecs: timeoutSecs})
	if err != nil {
		return "", 0, err
	}
	return out.Output, out.ExitCode, nil
}

// svcBadReq 参数类业务错误（直接给用户可读消息，不走 Wrap 以免追加通用后缀）。
func svcBadReq(msg string) *errs.Error {
	return errs.New(errs.CodeBadRequest, "error.badRequest", msg)
}

// buildB64WriteScript 生成「备份 → 原子替换」写文件脚本：内容经 base64 嵌入，
// 避免引号/换行/注入问题（路径必须是服务端常量，不得拼接用户输入）；
// 保留原文件权限与属主，写前备份 <path>.bak。
func buildB64WriteScript(path string, content []byte) string {
	b64 := base64.StdEncoding.EncodeToString(content)
	return fmt.Sprintf(`set -e
mkdir -p "$(dirname '%s')"
[ -f '%s' ] && cp -f '%s' '%s.bak' || true
_m=$(stat -c '%%a' '%s' 2>/dev/null || echo 644)
_o=$(stat -c '%%u:%%g' '%s' 2>/dev/null || echo 0:0)
printf '%%s' '%s' | base64 -d > '%s.tmp'
chmod "$_m" '%s.tmp'
chown "$_o" '%s.tmp'
mv -f '%s.tmp' '%s'
echo YPOK`,
		path, path, path, path, path, path, b64, path, path, path, path, path)
}
