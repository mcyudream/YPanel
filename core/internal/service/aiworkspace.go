package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ypanel/core/internal/agentclient"
	"github.com/ypanel/core/internal/rbac"
	"github.com/ypanel/shared/dto"
	"strings"
	"time"
)

func (s *AIService) WorkspaceFiles(ctx context.Context) (*json.RawMessage, error) {
	if err := rbac.CheckTool(ctx, aiModFiles, aiRiskRead, `{"node":"local"}`); err != nil {
		return nil, err
	}
	ac, err := s.acLocal()
	if err != nil {
		return nil, err
	}
	return agentclient.GetJSON[json.RawMessage](ac, ctx, "/agent/v1/files/list?path="+workspaceDir)
}

// The workspace is a host directory, not a security sandbox.
func (s *AIService) RunWorkspace(ctx context.Context, command string) (*dto.ExecResp, error) {
	node, _ := ctx.Value(aiNodeCtxKey{}).(string)
	if node == "" {
		node = "local"
	}
	scopeArgs, _ := json.Marshal(map[string]string{"node": node})
	if err := rbac.CheckTool(ctx, aiModExec, aiRiskDanger, string(scopeArgs)); err != nil {
		return nil, err
	}
	if s.settings != nil && NewSecuritySettingsService(s.settings).DangerLockEnabled() {
		return nil, errWrapAI("危险操作锁已开启")
	}
	if strings.TrimSpace(command) == "" || len(command) > 65536 {
		return nil, errWrapAI("命令为空或过长")
	}
	ac, err := s.acFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 125*time.Second)
	defer cancel()
	out, err := agentclient.DoJSON[dto.ExecReq, dto.ExecResp](ac, ctx, "POST", "/agent/v1/exec", &dto.ExecReq{Command: fmt.Sprintf("mkdir -p '%s' && cd '%s' && {\n%s\n}", workspaceDir, workspaceDir, command), TimeoutSecs: 120})
	args, _ := json.Marshal(map[string]string{"command": command, "node": node})
	result := ""
	if out != nil {
		result = out.Output
	}
	if err != nil {
		result = err.Error()
	}
	s.auditOperation(ctx, aiToolDef{Name: "run_in_workspace", Module: aiModExec, Risk: aiRiskDanger}, string(args), result, "", err == nil && out != nil && out.ExitCode == 0, "executed")
	return out, err
}
