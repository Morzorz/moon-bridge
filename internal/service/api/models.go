package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"moonbridge/internal/config"
)

// ---- Models ----

// GET /models
func (r *Router) handleListModels(w http.ResponseWriter, req *http.Request) {
	p := parsePagination(req)

	cfg := r.runtime.Current()

	type modelItem struct {
		Slug          string   `json:"slug"`
		DisplayName   string   `json:"display_name,omitempty"`
		ContextWindow int      `json:"context_window"`
		Providers     []string `json:"providers"`
	}

	slugs := make([]string, 0, len(cfg.Config.Models))
	for slug := range cfg.Config.Models {
		slugs = append(slugs, slug)
	}
	sortStrings(slugs)

	total := len(slugs)

	sliceEnd := p.Offset + p.Limit
	if p.Offset > len(slugs) {
		p.Offset = len(slugs)
	}
	if sliceEnd > len(slugs) {
		sliceEnd = len(slugs)
	}
	page := slugs[p.Offset:sliceEnd]

	// Build provider index for each model.
	modelProviders := make(map[string][]string)
	for pk, def := range cfg.Config.ProviderDefs {
		for _, offer := range def.Offers {
			modelProviders[offer.Model] = append(modelProviders[offer.Model], pk)
		}
	}

	items := make([]modelItem, 0, len(page))
	for _, slug := range page {
		def := cfg.Config.Models[slug]
		providers := modelProviders[slug]
		if providers == nil {
			providers = []string{}
		}
		items = append(items, modelItem{
			Slug:          slug,
			DisplayName:   def.DisplayName,
			ContextWindow: def.ContextWindow,
			Providers:     providers,
		})
	}

	respondJSON(w, http.StatusOK, paginatedResponse{
		Data:   items,
		Total:  total,
		Limit:  p.Limit,
		Offset: p.Offset,
	})
}

// GET /models/{slug}
func (r *Router) handleGetModel(w http.ResponseWriter, req *http.Request) {
	slug := req.PathValue("slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, "invalid_slug", "无效的 model slug")
		return
	}

	cfg := r.runtime.Current()
	def, ok := cfg.Config.Models[slug]
	if !ok {
		respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("model %q 不存在", slug))
		return
	}

	// Find which providers offer this model.
	providers := make([]string, 0)
	for pk, pdef := range cfg.Config.ProviderDefs {
		for _, offer := range pdef.Offers {
			if offer.Model == slug {
				providers = append(providers, pk)
				break
			}
		}
	}

	resp := map[string]any{
		"slug":               slug,
		"display_name":       def.DisplayName,
		"description":        def.Description,
		"context_window":     def.ContextWindow,
		"max_output_tokens":  def.MaxOutputTokens,
		"input_modalities":   def.InputModalities,
		"providers":          providers,
	}

	respondJSON(w, http.StatusOK, resp)
}

// PUT /models/{slug}
func (r *Router) handlePutModel(w http.ResponseWriter, req *http.Request) {
	slug := req.PathValue("slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, "invalid_slug", "无效的 model slug")
		return
	}

	var body struct {
		DisplayName     string `json:"display_name"`
		Description     string `json:"description"`
		ContextWindow   int    `json:"context_window"`
		MaxOutputTokens int    `json:"max_output_tokens"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		cfg.Models[slug] = config.ModelDef{
			ContextWindow:   body.ContextWindow,
			MaxOutputTokens: body.MaxOutputTokens,
			DisplayName:     body.DisplayName,
			Description:     body.Description,
		}
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "模型已保存并生效",
	})
}

// DELETE /models/{slug}
func (r *Router) handleDeleteModel(w http.ResponseWriter, req *http.Request) {
	slug := req.PathValue("slug")
	if slug == "" {
		respondError(w, http.StatusBadRequest, "invalid_slug", "无效的 model slug")
		return
	}

	cfg := r.runtime.Current()
	if _, ok := cfg.Config.Models[slug]; !ok {
		respondError(w, http.StatusNotFound, "not_found", fmt.Sprintf("model %q 不存在", slug))
		return
	}

	// Check if any provider offers this model — refuse if referenced.
	for pk, def := range cfg.Config.ProviderDefs {
		for _, offer := range def.Offers {
			if offer.Model == slug {
				respondError(w, http.StatusConflict, "referenced",
					fmt.Sprintf("model %q 仍被 provider %q 引用，无法删除", slug, pk))
				return
			}
		}
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		delete(cfg.Models, slug)
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("删除失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "模型已删除并生效",
	})
}
