package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"moonbridge/internal/config"
)

// ---- Offers ----

// POST /providers/{key}/offers
func (r *Router) handleCreateOffer(w http.ResponseWriter, req *http.Request) {
	providerKey := req.PathValue("key")
	if providerKey == "" {
		respondError(w, http.StatusBadRequest, "invalid_key", "无效的 provider key")
		return
	}

	var body struct {
		Model        string  `json:"model"`
		UpstreamName string  `json:"upstream_name"`
		Priority     int     `json:"priority"`
		InputPrice   float64 `json:"input_price"`
		OutputPrice  float64 `json:"output_price"`
		CacheWrite   float64 `json:"cache_write"`
		CacheRead    float64 `json:"cache_read"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}
	if body.Model == "" {
		respondError(w, http.StatusBadRequest, "validation_error", "model 不能为空")
		return
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		def, ok := cfg.ProviderDefs[providerKey]
		if !ok {
			return fmt.Errorf("provider %q 不存在", providerKey)
		}
		def.Offers = append(def.Offers, config.OfferEntry{
			Model:        body.Model,
			UpstreamName: body.UpstreamName,
			Priority:     body.Priority,
			Pricing: config.ModelPricing{
				InputPrice:      body.InputPrice,
				OutputPrice:     body.OutputPrice,
				CacheWritePrice: body.CacheWrite,
				CacheReadPrice:  body.CacheRead,
			},
		})
		cfg.ProviderDefs[providerKey] = def
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("保存失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "Offer 已创建并生效",
	})
}

// PATCH /providers/{key}/offers/{model}
func (r *Router) handleUpdateOffer(w http.ResponseWriter, req *http.Request) {
	providerKey := req.PathValue("key")
	modelSlug := req.PathValue("model")
	if providerKey == "" || modelSlug == "" {
		respondError(w, http.StatusBadRequest, "invalid_path", "路径格式无效")
		return
	}

	// Use pointer types for numeric and optional string fields so that
	// zero-value vs. unset can be distinguished (P1-7).
	var body struct {
		UpstreamName *string  `json:"upstream_name"`
		Priority     *int     `json:"priority"`
		InputPrice   *float64 `json:"input_price"`
		OutputPrice  *float64 `json:"output_price"`
		CacheWrite   *float64 `json:"cache_write"`
		CacheRead    *float64 `json:"cache_read"`
	}

	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
		return
	}

	// Read the current offer to fill in omitted fields.
	cfg := r.runtime.Current()
	currentOffer := findOffer(cfg.Config, providerKey, modelSlug)

	after := map[string]any{
		"provider_key": providerKey,
		"model_slug":   modelSlug,
	}

	if body.UpstreamName != nil {
		after["upstream_name"] = *body.UpstreamName
	} else if currentOffer != nil {
		after["upstream_name"] = currentOffer.UpstreamName
	} else {
		after["upstream_name"] = ""
	}

	if body.Priority != nil {
		after["priority"] = *body.Priority
	} else if currentOffer != nil {
		after["priority"] = currentOffer.Priority
	} else {
		after["priority"] = 0
	}

	if body.InputPrice != nil {
		after["input_price"] = *body.InputPrice
	} else if currentOffer != nil {
		after["input_price"] = currentOffer.Pricing.InputPrice
	} else {
		after["input_price"] = 0.0
	}

	if body.OutputPrice != nil {
		after["output_price"] = *body.OutputPrice
	} else if currentOffer != nil {
		after["output_price"] = currentOffer.Pricing.OutputPrice
	} else {
		after["output_price"] = 0.0
	}

	if body.CacheWrite != nil {
		after["cache_write"] = *body.CacheWrite
	} else if currentOffer != nil {
		after["cache_write"] = currentOffer.Pricing.CacheWritePrice
	} else {
		after["cache_write"] = 0.0
	}

	if body.CacheRead != nil {
		after["cache_read"] = *body.CacheRead
	} else if currentOffer != nil {
		after["cache_read"] = currentOffer.Pricing.CacheReadPrice
	} else {
		after["cache_read"] = 0.0
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		def, ok := cfg.ProviderDefs[providerKey]
		if !ok {
			return fmt.Errorf("provider %q 不存在", providerKey)
		}
		for i := range def.Offers {
			if def.Offers[i].Model == modelSlug {
				o := &def.Offers[i]
				if body.UpstreamName != nil {
					o.UpstreamName = *body.UpstreamName
				}
				if body.Priority != nil {
					o.Priority = *body.Priority
				}
				if body.InputPrice != nil {
					o.Pricing.InputPrice = *body.InputPrice
				}
				if body.OutputPrice != nil {
					o.Pricing.OutputPrice = *body.OutputPrice
				}
				if body.CacheWrite != nil {
					o.Pricing.CacheWritePrice = *body.CacheWrite
				}
				if body.CacheRead != nil {
					o.Pricing.CacheReadPrice = *body.CacheRead
				}
				cfg.ProviderDefs[providerKey] = def
				return nil
			}
		}
		return fmt.Errorf("offer %s/%s 不存在", providerKey, modelSlug)
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("更新失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "Offer 已更新并生效",
	})
}

// DELETE /providers/{key}/offers/{model}
func (r *Router) handleDeleteOffer(w http.ResponseWriter, req *http.Request) {
	providerKey := req.PathValue("key")
	modelSlug := req.PathValue("model")
	if providerKey == "" || modelSlug == "" {
		respondError(w, http.StatusBadRequest, "invalid_path", "路径格式无效")
		return
	}

	err := r.modifyConfig(func(cfg *config.Config) error {
		def, ok := cfg.ProviderDefs[providerKey]
		if !ok {
			return fmt.Errorf("provider %q 不存在", providerKey)
		}
		filtered := make([]config.OfferEntry, 0, len(def.Offers))
		for _, o := range def.Offers {
			if o.Model != modelSlug {
				filtered = append(filtered, o)
			}
		}
		if len(filtered) == len(def.Offers) {
			return fmt.Errorf("offer %s/%s 不存在", providerKey, modelSlug)
		}
		def.Offers = filtered
		cfg.ProviderDefs[providerKey] = def
		return nil
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "apply_error", fmt.Sprintf("删除失败: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "success",
		"message": "Offer 已删除并生效",
	})
}

// findOffer looks up an OfferEntry in the current config for the given provider+model.
func findOffer(cfg config.Config, providerKey, modelSlug string) *config.OfferEntry {
	def, ok := cfg.ProviderDefs[providerKey]
	if !ok {
		return nil
	}
	for i := range def.Offers {
		if def.Offers[i].Model == modelSlug {
			return &def.Offers[i]
		}
	}
	return nil
}
