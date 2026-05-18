package config

// ProxyConfig is the config domain model for the proxy layer.
type ProxyConfig struct {
	ResponseModel            string
	ResponseProviderBaseURL  string
	ResponseProviderAPIKey   string
	AnthropicModel           string
	AnthropicProviderBaseURL string
	AnthropicProviderAPIKey  string
	AnthropicProviderVersion string
}

// ProxyFromGlobalConfig extracts proxy-relevant fields from the global config.
func ProxyFromGlobalConfig(cfg *Config) ProxyConfig {
	return ProxyConfig{
		ResponseModel:            cfg.OpenAIProvider,
		ResponseProviderBaseURL:  "",
		ResponseProviderAPIKey:   "",
		AnthropicModel:           cfg.AnthropicProvider,
		AnthropicProviderBaseURL: "",
		AnthropicProviderAPIKey:  "",
		AnthropicProviderVersion: "",
	}
}
