package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"log/slog"
	"moonbridge/internal/config"
	"moonbridge/internal/db"
	"moonbridge/internal/logger"
	"moonbridge/internal/protocol/anthropic"
	"moonbridge/internal/protocol/google"
	"moonbridge/internal/protocol/chat"
	"moonbridge/internal/protocol/cache"
	"moonbridge/internal/format"
	"moonbridge/internal/protocol/openai"
	"moonbridge/internal/service/provider"
	"moonbridge/internal/service/proxy"
	"moonbridge/internal/service/runtime"
	"moonbridge/internal/service/server"
	"moonbridge/internal/service/server/session"
	"moonbridge/internal/service/server/trace"
	"moonbridge/internal/service/server/usage"
	"moonbridge/internal/service/stats"
	mbtrace "moonbridge/internal/service/trace"
)

const Name = "Moon Bridge"

func Run(output io.Writer) {
	fmt.Fprintln(output, WelcomeMessage())
}

func WelcomeMessage() string {
	return "欢迎使用 " + Name + "!"
}

func RunServer(ctx context.Context, cfg config.Config, errors io.Writer) error {
	slog.Info("启动服务器", "mode", "Unified", "addr", cfg.Addr)
	return runTransform(ctx, cfg, errors)
}

func runTransform(ctx context.Context, cfg config.Config, errors io.Writer) error {
	// Construct domain configs from global config.
	serverCfg := config.ServerFromGlobalConfig(&cfg)
	cacheCfg := config.CacheFromGlobalConfig(&cfg)
	proxyCfg := config.ProxyFromGlobalConfig(&cfg)
	storeCfg := config.StoreFromGlobalConfig(&cfg)
	persistCfg := config.PersistenceFromGlobalConfig(&cfg)
	providerCfg := config.ProviderFromGlobalConfig(&cfg)
	_ = persistCfg   // used in db init
	_ = storeCfg     // used in config store
	_ = proxyCfg     // used in proxy mode

		// === Phase 1: Bootstrap from YAML ===

	// Build multi-provider infrastructure from YAML config.
	providerDefs := provider.BuildProviderDefsFromConfig(providerCfg)
	modelRoutes := provider.BuildModelRoutesFromConfig(providerCfg)
	providerMgr, err := provider.NewProviderManager(providerDefs, modelRoutes)
	if err != nil {
		return fmt.Errorf("init provider manager: %w", err)
	}

	// Resolve a fallback client for web search probing and server fallback.
	defaultClient := resolveDefaultClient(providerMgr, errors)
	resolvePerProviderWebSearch(ctx, cfg, providerMgr, errors)

	sessionStats := stats.NewSessionStats()
	pricing := provider.BuildPricingFromConfig(providerCfg)
	if len(pricing) > 0 {
		sessionStats.SetPricing(pricing)
	}

	tracer := mbtrace.New(mbtrace.Config{
		Enabled: cfg.TraceRequests,
		Root:    transformTraceRoot(),
	})
	logTrace(errors, "transform", tracer)

	// Determine the default provider to use as the fallback Provider.
	var fallbackProvider provider.ProviderClient
	if defaultClient != nil {
		fallbackProvider = provider.NewAnthropicClientAdapter(defaultClient)
	}

	// Register plugins.
	plugins := BuiltinExtensions().NewRegistry(slog.Default(), cfg)
	if err := plugins.InitAll(&cfg); err != nil {
		return fmt.Errorf("init plugins: %w", err)
	}
	defer plugins.ShutdownAll()

	// Wire plugin LogConsumer into the slog consume pipeline.
	logger.SetConsumeFunc(func(entries []logger.LogEntry) []logger.LogEntry {
		return plugins.ConsumeGlobalLog(entries)
	})
	logger.CaptureLogToBuffer()

	// Initialize persistence layer (db.Registry).
	dbRegistry := db.NewRegistry(slog.Default())
	for _, p := range plugins.DBProviders() {
		if prov := p.DBProvider(); prov != nil {
			dbRegistry.RegisterProvider(prov)
		}
	}
	for _, c := range plugins.DBConsumers() {
		if cons := c.DBConsumer(); cons != nil {
			dbRegistry.RegisterConsumer(cons)
		}
	}
	if err := dbRegistry.Init(ctx, cfg.Persistence.ActiveProvider); err != nil {
		return fmt.Errorf("init persistence: %w", err)
	}
	defer dbRegistry.Shutdown()

		// === Phase 3: Build Runtime ===
	rt := runtime.NewRuntime(cfg, providerMgr, pricing)

	// === Phase 4: Build Server with Runtime ===
	// Create shared cache registry (used by both Bridge and Adapter paths).
	cacheReg := cache.NewMemoryRegistry()

	// Optionally create the experimental adapter registry.
	// Create the adapter registry for Core format dispatch.
	adapterReg := format.NewRegistry()
	coreHooks := plugins.CorePluginHooks()

	// Inbound: OpenAI Responses client adapter.
	oaiAdapter := openai.NewOpenAIAdapter(coreHooks)
	_ = adapterReg.RegisterClient(oaiAdapter)
	_ = adapterReg.RegisterClientStream(oaiAdapter)

	// Upstream: Anthropic provider adapter with cache manager.
	cacheMgr := anthropic.NewCacheManager(&cfg.Cache, cacheReg)
	anthAdapter := anthropic.NewAnthropicProviderAdapter(cfg.DefaultMaxTokens, cacheMgr, coreHooks)
	_ = adapterReg.RegisterProvider(anthAdapter)
	_ = adapterReg.RegisterProviderStream(anthAdapter)

	// Upstream: Google GenAI provider adapter.
		googleCfg := &cache.PlanCacheConfig{
			Mode:                     cacheCfg.Mode,
			TTL:                      cacheCfg.TTL,
			PromptCaching:            cacheCfg.PromptCaching,
			AutomaticPromptCache:     cacheCfg.AutomaticPromptCache,
			ExplicitCacheBreakpoints: cacheCfg.ExplicitCacheBreakpoints,
			AllowRetentionDowngrade:  cacheCfg.AllowRetentionDowngrade,
			MaxBreakpoints:           cacheCfg.MaxBreakpoints,
			MinCacheTokens:           cacheCfg.MinCacheTokens,
			ExpectedReuse:            cacheCfg.ExpectedReuse,
			MinimumValueScore:        cacheCfg.MinimumValueScore,
			MinBreakpointTokens:      cacheCfg.MinBreakpointTokens,
		}
	googleAdapter := google.NewGeminiProviderAdapter(cfg.DefaultMaxTokens, nil, coreHooks, googleCfg, cacheReg)
	_ = adapterReg.RegisterProvider(googleAdapter)
	_ = adapterReg.RegisterProviderStream(googleAdapter)

	// Upstream: OpenAI Chat provider adapter.
	chatAdapter := chat.NewChatProviderAdapter(cfg.DefaultMaxTokens, nil, coreHooks)
	_ = adapterReg.RegisterProvider(chatAdapter)
	_ = adapterReg.RegisterProviderStream(chatAdapter)

	slog.Info("Adapter dispatch path enabled", "registry", "format.Registry")

	// Build protocol-specific HTTP clients from provider configs.
	chatClients := make(map[string]any, len(cfg.ProviderDefs))
	googleClients := make(map[string]any, len(cfg.ProviderDefs))
	for key, def := range cfg.ProviderDefs {
		switch def.Protocol {
		case config.ProtocolOpenAIChat:
			chatClients[key] = chat.NewClient(chat.ClientConfig{
				BaseURL:   def.BaseURL,
				APIKey:    def.APIKey,
				UserAgent: def.UserAgent,
			})
			slog.Debug("chat client created", "provider", key)
		case config.ProtocolGoogleGenAI:
			googleClients[key] = google.NewClient(google.ClientConfig{
				BaseURL:   def.BaseURL,
				APIKey:    def.APIKey,
				Project:   def.Project,
				Location:  def.Location,
				Version:   def.APIVersion,
				UserAgent: def.UserAgent,
			})
			slog.Debug("google client created", "provider", key)
		}
	}


	// Optional: create proxy handlers for transparent proxy endpoints.
	var openaiProxyHandler, anthropicProxyHandler http.Handler
	if cfg.HasOpenAIProxy() {
		def := cfg.ProviderDefs[cfg.OpenAIProvider]
		p, err := proxy.NewResponse(proxy.ResponseConfig{
			UpstreamBaseURL: def.BaseURL,
			APIKey:          def.APIKey,
			Tracer:          tracer,
			TraceErrors:     errors,
			IsEnabled: func() bool {
				return rt.Current().Config.HasOpenAIProxy()
			},
			ModelMap: cfg.OpenAIProxyModelMap,
		})
		if err != nil {
			slog.Warn("OpenAI 代理初始化失败，已禁用", "error", err)
		} else {
			openaiProxyHandler = p
			slog.Info("OpenAI 透明代理已就绪", "provider", cfg.OpenAIProvider)
		}
	}
	if cfg.HasAnthropicProxy() {
		def := cfg.ProviderDefs[cfg.AnthropicProvider]
		p, err := proxy.NewAnthropic(proxy.AnthropicConfig{
			UpstreamBaseURL: def.BaseURL,
			APIKey:          def.APIKey,
			Version:         def.Version,
			Tracer:          tracer,
			TraceErrors:     errors,
			IsEnabled: func() bool {
				return rt.Current().Config.HasAnthropicProxy()
			},
			ModelMap: cfg.AnthropicProxyModelMap,
		})
		if err != nil {
			slog.Warn("Anthropic 代理初始化失败，已禁用", "error", err)
		} else {
			anthropicProxyHandler = p
			slog.Info("Anthropic 透明代理已就绪", "provider", cfg.AnthropicProvider)
		}
	}

	// Create sub-package managers for session, usage, and trace.
	sessMgr := session.NewInMemoryManager(server.NewSessionConfigAdapter(serverCfg), plugins)
	usageTrk := usage.NewStatsTracker(sessionStats)
	traceWtr := trace.NewFileWriter(tracer, errors)

	handler := server.New(server.Config{
		ServerCfg:      serverCfg,
		Provider:        fallbackProvider,
		ProviderMgr:     providerMgr,
		ChatClients:     chatClients,
		GoogleClients:   googleClients,
		Tracer:          tracer,
		TraceErrors:     errors,
		Stats:           sessionStats,
		PluginRegistry:  plugins,
		AppConfig:       serverCfg,
		Runtime:         rt,
		AdapterRegistry: adapterReg,
		SessionManager:  sessMgr,
		UsageTracker:    usageTrk,
		TraceWriter:     traceWtr,
		OpenAIProxyHandler:   openaiProxyHandler,
		AnthropicProxyHandler: anthropicProxyHandler,
	})

	wrapped := handler
	return runHTTPServer(ctx, cfg.Addr, wrapped, errors, sessionStats)
}

// resolveDefaultClient returns the provider client for the default key.
// Returns nil when no default provider is configured (all models use explicit routing).
func resolveDefaultClient(pm *provider.ProviderManager, errors io.Writer) *anthropic.Client {
	if pm.DefaultKey() == "" {
		slog.Warn("未配置默认提供商，跳过网页搜索探测和服务器回退")
		return nil
	}
	client, err := pm.ClientForKey(pm.DefaultKey())
	if err != nil {
		slog.Warn("默认提供商客户端不可用", "error", err)
		return nil
	}
	if acc, ok := client.(provider.AnthropicClientAccessor); ok {
		return acc.AnthropicClient()
	}
	slog.Warn("默认提供商客户端不支持访问底层客户端")
	return nil
}

// webSearchProber interface and following functions are unchanged.
type webSearchProber interface {
	ProbeWebSearch(context.Context, string) (bool, error)
}

// resolvePerProviderWebSearch resolves web_search support for each provider and
// each model that has a model-level override.
func resolvePerProviderWebSearch(ctx context.Context, cfg config.Config, pm *provider.ProviderManager, errors io.Writer) {
	if pm == nil {
		return
	}
	// 1. Resolve provider-level defaults.
	for _, key := range pm.ProviderKeys() {
		protocol := pm.ProtocolForKey(key)
		support := cfg.WebSearchForProvider(key)
		switch protocol {
		case config.ProtocolAnthropic:
			switch support {
			case config.WebSearchSupportDisabled:
				pm.SetResolvedWebSearch(key, "disabled")
				slog.Info("配置禁用网页搜索", "provider", key)
			case config.WebSearchSupportEnabled:
				pm.SetResolvedWebSearch(key, "enabled")
				slog.Info("配置强制启用网页搜索", "provider", key)
			case config.WebSearchSupportInjected:
				pm.SetResolvedWebSearch(key, "injected")
				slog.Info("网页搜索注入模式已启用", "provider", key)
			default:
				resolved := probeProviderWebSearch(ctx, key, pm, errors)
				if resolved == "disabled" && cfg.TavilyAPIKey != "" {
					resolved = "injected"
					slog.Info("网页搜索自动探测失败，回退到注入模式", "provider", key)
				}
				pm.SetResolvedWebSearch(key, resolved)
			}
		case config.ProtocolOpenAIResponse:
			switch support {
			case config.WebSearchSupportDisabled, config.WebSearchSupportInjected:
				pm.SetResolvedWebSearch(key, "disabled")
				slog.Info("响应端网页搜索已禁用", "provider", key, "protocol", protocol, "config", support)
			default:
				pm.SetResolvedWebSearch(key, "enabled")
				slog.Info("已启用响应端网页搜索", "provider", key, "protocol", protocol)
			}
		default:
			// openai-chat 和 google-genai 无原生 web_search，有 API key 时启用注入模式
			if cfg.TavilyAPIKey != "" {
				pm.SetResolvedWebSearch(key, "injected")
				slog.Info("注入式网页搜索已启用", "provider", key, "protocol", protocol)
			} else {
				pm.SetResolvedWebSearch(key, "disabled")
				slog.Info("跳过网页搜索：无 Tavily API key", "provider", key, "protocol", protocol)
			}
		}
	}
	// 2. Resolve model-level overrides for provider catalog slugs and route aliases.
	for providerKey, def := range cfg.ProviderDefs {
		providerWS := cfg.WebSearchForProvider(providerKey)
		for modelName := range def.Models {
			alias := providerKey + "/" + modelName
			newAlias := modelName + "(" + providerKey + ")"
			modelWS := cfg.WebSearchForModel(alias)
			resolveModelWebSearch(ctx, alias, modelWS, providerWS, pm, errors)
			resolveModelWebSearch(ctx, newAlias, modelWS, providerWS, pm, errors)
			pureWS := cfg.WebSearchForModel(modelName)
			resolveModelWebSearch(ctx, modelName, pureWS, providerWS, pm, errors)
		}
	}
	for alias, route := range cfg.Routes {
		modelWS := cfg.WebSearchForModel(alias)
		providerWS := cfg.WebSearchForProvider(route.Provider)
		resolveModelWebSearch(ctx, alias, modelWS, providerWS, pm, errors)
	}
}

func resolveModelWebSearch(ctx context.Context, alias string, modelWS config.WebSearchSupport, providerWS config.WebSearchSupport, pm *provider.ProviderManager, errors io.Writer) {
	if modelWS == providerWS {
		return
	}
	modelKey := "model:" + alias
	protocol := pm.ProtocolForModel(alias)
	switch protocol {
	case config.ProtocolAnthropic:
	case config.ProtocolOpenAIResponse:
		switch modelWS {
		case config.WebSearchSupportDisabled, config.WebSearchSupportInjected:
			pm.SetResolvedWebSearch(modelKey, "disabled")
			slog.Info("模型禁用响应端网页搜索", "model", alias, "config", modelWS)
		default:
			pm.SetResolvedWebSearch(modelKey, "enabled")
			slog.Info("模型启用响应端网页搜索", "model", alias)
		}
		return
	default:
		pm.SetResolvedWebSearch(modelKey, "disabled")
		slog.Info("跳过模型级网页搜索：不支持的协议", "model", alias, "protocol", protocol)
		return
	}
	switch modelWS {
	case config.WebSearchSupportDisabled:
		pm.SetResolvedWebSearch(modelKey, "disabled")
		slog.Info("模型配置禁用网页搜索", "model", alias)
	case config.WebSearchSupportEnabled:
		pm.SetResolvedWebSearch(modelKey, "enabled")
		slog.Info("模型配置强制启用网页搜索", "model", alias)
	case config.WebSearchSupportInjected:
		pm.SetResolvedWebSearch(modelKey, "injected")
		slog.Info("模型配置启用网页搜索注入模式", "model", alias)
	default:
		resolved := probeModelWebSearch(ctx, alias, pm, errors)
		pm.SetResolvedWebSearch(modelKey, resolved)
	}
}

func probeProviderWebSearch(ctx context.Context, key string, pm *provider.ProviderManager, errors io.Writer) string {
	pc, err := pm.ClientForKey(key)
	if err != nil {
		slog.Warn("网页搜索探测跳过：客户端不可用", "provider", key, "error", err)
		return "disabled"
	}

	upstreamModel := pm.FirstUpstreamModelForKey(key)
	if upstreamModel == "" {
		slog.Warn("网页搜索自动探测跳过：无模型路由到提供商", "provider", key)
		return "disabled"
	}

	acc, ok := pc.(provider.AnthropicClientAccessor)
	if !ok {
		slog.Warn("网页搜索探测跳过：客户端不支持访问", "provider", key)
		return "disabled"
	}
	client := acc.AnthropicClient()
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	supported, err := client.ProbeWebSearch(probeCtx, upstreamModel)
	if err != nil {
		slog.Warn("网页搜索自动探测失败", "provider", key, "error", err)
		fmt.Fprintf(errors, "网页搜索自动探测失败（提供商 %s）: %v\n", key, err)
		return "disabled"
	}
	if !supported {
		slog.Warn("提供商不支持网页搜索", "provider", key, "model", upstreamModel)
		fmt.Fprintf(errors, "提供商 %s 不支持网页搜索\n", key)
		return "disabled"
	}
	slog.Info("提供商支持网页搜索", "provider", key, "model", upstreamModel)
	return "enabled"
}

func probeModelWebSearch(ctx context.Context, modelAlias string, pm *provider.ProviderManager, errors io.Writer) string {
	upstreamModel, pc, err := pm.ClientFor(modelAlias)
	if err != nil {
		slog.Warn("网页搜索模型探测跳过：客户端不可用", "model", modelAlias, "error", err)
		return "disabled"
	}
	acc, ok := pc.(provider.AnthropicClientAccessor)
	if !ok {
		slog.Warn("网页搜索模型探测跳过：客户端不支持访问", "model", modelAlias)
		return "disabled"
	}
	client := acc.AnthropicClient()
	if err != nil {
		slog.Warn("网页搜索模型探测跳过：客户端不可用", "model", modelAlias, "error", err)
		return "disabled"
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	supported, err := client.ProbeWebSearch(probeCtx, upstreamModel)
	if err != nil {
		slog.Warn("网页搜索模型探测失败", "model", modelAlias, "error", err)
		fmt.Fprintf(errors, "网页搜索模型探测失败（%s）: %v\n", modelAlias, err)
		return "disabled"
	}
	if !supported {
		slog.Warn("模型不支持网页搜索", "model", modelAlias)
		fmt.Fprintf(errors, "模型 %s 不支持网页搜索\n", modelAlias)
		return "disabled"
	}
	slog.Info("模型支持网页搜索", "model", modelAlias)
	return "enabled"
}

func logTrace(errors io.Writer, label string, tracer *mbtrace.Tracer) {
	if !tracer.Enabled() {
		fmt.Fprintf(errors, "%s 跟踪已禁用\n", label)
		return
	}
	slog.Info("跟踪已启用", "label", label, "dir", tracer.Directory())
	fmt.Fprintf(errors, "%s 跟踪已启用于 %s\n", label, tracer.Directory())
}

func transformTraceRoot() string {
	return filepath.Join(mbtrace.DefaultRoot, "Transform")
}

func captureResponseTraceConfig(enabled bool) mbtrace.Config {
	return mbtrace.Config{
		Enabled: enabled,
		Root:    filepath.Join(mbtrace.DefaultRoot, "Capture", "Response"),
	}
}

func captureAnthropicTraceConfig(enabled bool) mbtrace.Config {
	return mbtrace.Config{
		Enabled: enabled,
		Root:    filepath.Join(mbtrace.DefaultRoot, "Capture", "Anthropic"),
	}
}

func runHTTPServer(ctx context.Context, addr string, handler http.Handler, errors io.Writer, sessionStats *stats.SessionStats) error {
	httpServer := &http.Server{Addr: addr, Handler: handler}
	defer func() {
		if closer, ok := handler.(io.Closer); ok {
			_ = closer.Close()
		}
	}()
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(errors, "%s 监听于 %s\n", Name, addr)
		slog.Info("HTTP 服务器监听中", "addr", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		if sessionStats != nil {
			summary := sessionStats.Summary()
			slog.Info(stats.FormatSummaryLine(summary))
			fmt.Fprintln(errors)
			stats.WriteSummary(errors, summary)
		}
		shutdownCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		slog.Error("HTTP 服务器错误", "error", err)
		return err
	}
}

// DumpConfigSchema dumps JSON Schema files alongside the config file,
// including known plugin config types. Call via --dump-config-schema flag.
func DumpConfigSchema(configPath string) error {
	return config.DumpConfigSchemaWithOptions(configPath, config.SchemaOptions{
		ExtensionSpecs: BuiltinExtensions().ConfigSpecs(),
	})
}
