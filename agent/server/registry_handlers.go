// Docker registry 凭据管理（B10）：读写 /root/.docker/config.json 的 auths 段。
package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ypanel/shared/errs"
)

const dockerConfigPath = "/root/.docker/config.json"

type registryEntry struct {
	Registry string `json:"registry"`
	Username string `json:"username"`
	Auth     string `json:"auth"`
}

func readDockerAuths() (map[string]any, error) {
	raw, err := os.ReadFile(dockerConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var cfg struct {
		Auths map[string]any `json:"auths"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if cfg.Auths == nil {
		cfg.Auths = map[string]any{}
	}
	return cfg.Auths, nil
}

func writeDockerAuths(auths map[string]any) error {
	cfg := map[string]any{"auths": auths}
	// 保留其余顶层字段
	if raw, err := os.ReadFile(dockerConfigPath); err == nil {
		var existing map[string]any
		if json.Unmarshal(raw, &existing) == nil {
			for k, v := range existing {
				if k != "auths" {
					cfg[k] = v
				}
			}
		}
	}
	if dir := filepath.Dir(dockerConfigPath); dir != "" {
		_ = os.MkdirAll(dir, 0o700)
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dockerConfigPath, b, 0o600)
}

// handleRegistryList GET /agent/v1/docker/registry
func (s *Server) handleRegistryList(w http.ResponseWriter, r *http.Request) {
	auths, err := readDockerAuths()
	if err != nil {
		writeErr(w, errs.Wrap(errs.ErrBadRequest, err.Error()))
		return
	}
	list := []registryEntry{}
	for registry, v := range auths {
		entry := registryEntry{Registry: registry}
		if m, ok := v.(map[string]any); ok {
			if auth, ok := m["auth"].(string); ok {
				entry.Auth = auth
				if dec, err := base64.StdEncoding.DecodeString(auth); err == nil {
					parts := strings.SplitN(string(dec), ":", 2)
					if len(parts) > 0 {
						entry.Username = parts[0]
					}
				}
			}
		}
		list = append(list, entry)
	}
	writeOK(w, list)
}

// handleRegistrySet PUT /agent/v1/docker/registry {registry, username, password}
func (s *Server) handleRegistrySet(w http.ResponseWriter, r *http.Request) {
	req, err := decodeBody[struct {
		Registry  string `json:"registry" binding:"required"`
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
	}](r)
	if err != nil {
		writeErr(w, err)
		return
	}
	auths, err := readDockerAuths()
	if err != nil {
		writeErr(w, err)
		return
	}
	auth := base64.StdEncoding.EncodeToString([]byte(req.Username + ":" + req.Password))
	auths[req.Registry] = map[string]string{
		"auth":  auth,
		"email": req.Username,
	}
	if err := writeDockerAuths(auths); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}

// handleRegistryRemove DELETE /agent/v1/docker/registry?registry=
func (s *Server) handleRegistryRemove(w http.ResponseWriter, r *http.Request) {
	registry := r.URL.Query().Get("registry")
	if registry == "" {
		writeErr(w, errs.ErrBadRequest)
		return
	}
	auths, err := readDockerAuths()
	if err != nil {
		writeErr(w, err)
		return
	}
	delete(auths, registry)
	if err := writeDockerAuths(auths); err != nil {
		writeErr(w, err)
		return
	}
	writeOKEmpty(w)
}
