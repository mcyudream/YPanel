package service

import (
	"context"
	"encoding/json"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/ypanel/core/internal/model"
	"github.com/ypanel/core/internal/rbac"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAIMCPBridgeApprovalAndCache(t *testing.T) {
	s := newAIChatTest(t)
	remote := mcpserver.NewMCPServer("local-test", "1.0", mcpserver.WithToolCapabilities(false))
	remote.AddTool(mcp.NewTool("echo", mcp.WithDescription("test tool")), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("MCP bridge ok"), nil
	})
	server := httptest.NewServer(mcpserver.NewStreamableHTTPServer(remote))
	t.Cleanup(server.Close)
	t.Cleanup(s.CloseAllMCP)
	conf := MCPServerConf{Name: "mock", Transport: "streamable-http", URL: server.URL, Enabled: true, AllowPrivate: true}
	if err := s.SaveMCPServers([]MCPServerConf{conf}); err != nil {
		t.Fatal(err)
	}
	admin := rbac.WithCaller(context.Background(), rbac.Caller{UserID: 1, PermSet: map[string]struct{}{"*": {}}, AllNodes: true})
	admin = context.WithValue(admin, aiRunContextKey{}, aiRunIdentity{UserID: 1, ConversationID: 8, RunID: "mcp-test"})
	defs := s.aiMCPTools(admin)
	if len(defs) != 1 {
		t.Fatalf("MCP tools were not bridged: %d", len(defs))
	}
	if _, err := defs[0].Fn(aiTestUser(2), "{}"); err == nil {
		t.Fatal("MCP bypassed admin permission")
	}
	asks := make(chan string, 1)
	result := make(chan string, 1)
	go func() {
		out, err := s.governToolCall(admin, defs[0], "{}", func(ev any) {
			raw, _ := json.Marshal(ev)
			var frame struct {
				Ask *struct {
					ID string `json:"id"`
				} `json:"ask"`
			}
			json.Unmarshal(raw, &frame)
			if frame.Ask != nil {
				asks <- frame.Ask.ID
			}
		}, func() {}, aiModeFull)
		if err != nil {
			result <- err.Error()
		} else {
			result <- out
		}
	}()
	select {
	case id := <-asks:
		if !s.ResolveAsk(id, true, admin) {
			t.Fatal("approval failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("full mode auto-approved unknown external risk")
	}
	select {
	case out := <-result:
		if !strings.Contains(out, "MCP bridge ok") {
			t.Fatal(out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("MCP call timed out")
	}
	var logs []model.AIOperationLog
	s.db.Find(&logs)
	if len(logs) == 0 || logs[0].UserID != 1 || logs[0].ConversationID != 8 || logs[0].RunID != "mcp-test" {
		t.Fatal("MCP audit identity missing")
	}
	if err := s.SaveMCPServers(nil); err != nil {
		t.Fatal(err)
	}
	if _, exists := s.mcpSessions.Load(mcpSessionKey(conf.Name)); exists {
		t.Fatal("configuration change retained stale connection")
	}
}
