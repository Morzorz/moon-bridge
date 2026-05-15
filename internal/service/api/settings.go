package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"moonbridge/internal/config"

	"gopkg.in/yaml.v3"
)

// ---- Settings ----

// GET /defaults
func (r *Router) handleGetDefaults(w http.ResponseWriter, req *http.Request) {
	cfg := r.runtime.Current()

	resp := map[string]any{
		"model":         cfg.Config.Defaults.Model,
		"max_tokens":    cfg.Config.Defaults.MaxTokens,
		"system_prompt": cfg.Config.Defaults.SystemPrompt,
	}

	respondJSON(w, http.StatusOK, resp)
}

// PUT /defaults
func (r *Router) handlePutDefaults(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Model        string `json:"model"`
		MaxTokens    int    `json:"max_tokens"`
		SystemPrompt string `json:"system_prompt"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		cfg.Defaults.Model = body.Model
		cfg.Defaults.MaxTokens = body.MaxTokens
		cfg.Defaults.SystemPrompt = body.SystemPrompt
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "默认参数已保存并生效",
	})
}

// GET /settings/mode
func (r *Router) handleGetMode(w http.ResponseWriter, req *http.Request) {
	cfg := r.runtime.Current()
	respondJSON(w, http.StatusOK, map[string]any{
		"mode": string(cfg.Config.Mode),
	})
}

// PUT /settings/mode
func (r *Router) handlePutMode(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}
	switch body.Mode {
	case "Transform", "CaptureResponse", "CaptureAnthropic":
	default:
		respondError(w, http.StatusBadRequest, "invalid_mode", "mode 必须是 Transform / CaptureResponse / CaptureAnthropic 之一")
		return
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		cfg.Mode = config.Mode(body.Mode)
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "模式已切换（需重启服务器完全生效）",
	})
}

// ---- Capabilities ----

// GET /capabilities
func (r *Router) handleGetCapabilities(w http.ResponseWriter, req *http.Request) {
	cfg := r.runtime.Current()
	respondJSON(w, http.StatusOK, map[string]any{
		"transform":       true,
		"proxy_openai":    cfg.Config.HasOpenAIProxy(),
		"proxy_anthropic": cfg.Config.HasAnthropicProxy(),
	})
}

// PUT /capabilities/proxy/response
func (r *Router) handlePutResponseProxyEnabled(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}
	err := r.modifyConfig(func(cfg *config.Config) error {
		cfg.ResponseProxy.Enabled = body.Enabled
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "代理开关已更改",
	})
}

// PUT /capabilities/proxy/anthropic
func (r *Router) handlePutAnthropicProxyEnabled(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}
	err := r.modifyConfig(func(cfg *config.Config) error {
		cfg.AnthropicProxy.Enabled = body.Enabled
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "代理开关已更改",
	})
}

// GET /web-search
func (r *Router) handleGetWebSearch(w http.ResponseWriter, req *http.Request) {
	cfg := r.runtime.Current()

	resp := map[string]any{
		"support":            string(cfg.Config.WebSearchSupport),
		"max_uses":           cfg.Config.WebSearchMaxUses,
		"tavily_api_key":     maskAPIKey(cfg.Config.TavilyAPIKey),
		"firecrawl_api_key":  maskAPIKey(cfg.Config.FirecrawlAPIKey),
		"search_max_rounds":  cfg.Config.SearchMaxRounds,
	}

	respondJSON(w, http.StatusOK, resp)
}

// PUT /web-search
func (r *Router) handlePutWebSearch(w http.ResponseWriter, req *http.Request) {
	var body struct {
		Support         string `json:"support"`
		MaxUses         int    `json:"max_uses"`
		TavilyAPIKey    string `json:"tavily_api_key"`
		FirecrawlAPIKey string `json:"firecrawl_api_key"`
		SearchMaxRounds int    `json:"search_max_rounds"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}

	// If masked, keep existing values.
	cfg := r.runtime.Current()
	tavilyKey := body.TavilyAPIKey
	if tavilyKey == "******" {
		tavilyKey = cfg.Config.TavilyAPIKey
	}
	firecrawlKey := body.FirecrawlAPIKey
	if firecrawlKey == "******" {
		firecrawlKey = cfg.Config.FirecrawlAPIKey
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		cfg.WebSearchSupport = config.WebSearchSupport(body.Support)
		cfg.WebSearchMaxUses = body.MaxUses
		cfg.TavilyAPIKey = tavilyKey
		cfg.FirecrawlAPIKey = firecrawlKey
		cfg.SearchMaxRounds = body.SearchMaxRounds
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "Web Search 配置已保存并生效",
	})
}

// GET /extensions
func (r *Router) handleListExtensions(w http.ResponseWriter, req *http.Request) {
	if r.registry == nil {
		respondJSON(w, http.StatusOK, []any{})
		return
	}

	names := r.registry.Plugins()
	respondJSON(w, http.StatusOK, names)
}

// GET /extensions/{name}
func (r *Router) handleGetExtension(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "invalid_name", "无效的 extension name")
		return
	}

	if r.registry == nil {
		respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("extension %q 不存在", name))
		return
	}

	ext := r.registry.Plugin(name)
	if ext == nil {
		respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("extension %q 不存在", name))
		return
	}

	respondJSON(w, http.StatusOK, ext)
}

// PUT /extensions/{name}
func (r *Router) handlePutExtension(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "invalid_name", "无效的 extension name")
		return
	}

	var body map[string]any
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}

	respondError(w, http.StatusNotImplemented, "not_supported", "Extension 配置请在 config.yml 中编辑")
}

// ---- Config ----

// GET /config/effective
func (r *Router) handleGetConfigEffective(w http.ResponseWriter, req *http.Request) {
	cfg := r.runtime.Current()
	fc := cfg.Config.ToFileConfig()
	maskFileConfigSecrets(&fc)
	respondJSON(w, http.StatusOK, fc)
}

// GET /config/export
func (r *Router) handleGetConfigExport(w http.ResponseWriter, req *http.Request) {
	cfg := r.runtime.Current()
	fc := cfg.Config.ToFileConfig()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(fc); err != nil {
		respondError(w, http.StatusInternalServerError, "export_error", fmt.Sprintf("序列化失败: %v", err))
		return
	}
	enc.Close()

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=moonbridge-config-%s.yml", time.Now().Format("20060102-150405")))
	w.Write(buf.Bytes())
}

// POST /config/import
func (r *Router) handlePostConfigImport(w http.ResponseWriter, req *http.Request) {
	var body struct {
		YAML string `json:"yaml"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}
	if body.YAML == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "yaml 不能为空")
		return
	}

	// Parse, validate, save directly, and reload.
	cfg, err := config.LoadFromYAML([]byte(body.YAML))
	if err != nil {
		respondError(w, http.StatusBadRequest, "parse_error", fmt.Sprintf("YAML 解析失败: %v", err))
		return
	}
	if err := cfg.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, "validation_error", fmt.Sprintf("配置校验失败: %v", err))
		return
	}
	if err := cfg.Save(); err != nil {
		respondError(w, http.StatusInternalServerError, "save_error", fmt.Sprintf("保存失败: %v", err))
		return
	}
	if err := r.runtime.Reload(cfg); err != nil {
		respondError(w, http.StatusInternalServerError, "reload_error", fmt.Sprintf("重载失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "配置已导入并生效",
	})
}

// POST /config/validate
func (r *Router) handlePostConfigValidate(w http.ResponseWriter, req *http.Request) {
	var body struct {
		ConfigJSON string `json:"config"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}
	if body.ConfigJSON == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "config 不能为空")
		return
	}

	_, err := config.LoadFromYAML([]byte(body.ConfigJSON))
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]any{
			"valid":  false,
			"errors": []string{err.Error()},
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"valid": true,
	})
}

// ---- Changes ----

// GET /changes
// Changes API endpoints are removed. Config changes now take effect immediately.
// See individual PUT handlers for direct config.yml modification.
