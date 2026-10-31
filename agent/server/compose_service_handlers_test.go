package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ypanel/agent/internal/compose"
	"github.com/ypanel/shared/dto"
	"github.com/ypanel/shared/errs"
)

func TestComposeServiceActionHandler(t *testing.T) {
	bin, external := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(external, "compose.yaml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := compose.New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{"project": {"demo"}, "service": {"web"}, "action": {"restart"}, "dir": {external}}
	for _, disabled := range []bool{false, true} {
		s := &Server{compose: m}
		if disabled {
			s.compose = nil
		}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/agent/v1/compose/service-action?"+query.Encode(), nil)
		s.handleComposeServiceAction(w, r)
		var out struct {
			Code int                          `json:"code"`
			Data dto.ComposeServiceActionResp `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if disabled {
			if out.Code != errs.CodeAgentDisabled {
				t.Fatalf("能力不可用没有降级: %s", w.Body.String())
			}
		} else if out.Code != 0 || !strings.Contains(out.Data.Output, external) || !strings.HasSuffix(out.Data.Output, "restart\nweb\n") {
			t.Fatalf("目录或服务操作没有传给 compose: %s", w.Body.String())
		}
	}
}
