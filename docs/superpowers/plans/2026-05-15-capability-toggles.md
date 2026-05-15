# 三种模式并存 + 能力开关 GUI 重构 — 实现计划

> **For agentic workers:** Use executing-plans skill. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Moon Bridge 的三种互斥工作模式重构为「能力开关」架构：Transform 协议转换引擎常驻运行，OpenAI/Anthropic 透明代理通道作为可独立启停的附加功能，GUI 同步从模式选择器改为能力开关面板。

**Architecture:** 后端始终启动完整 Transform 服务器，在 `Server.New()` 中根据 `proxy.response.enabled` 和 `proxy.anthropic.enabled` 配置可选地挂载两个透明代理 HTTP handler 到 mux。前端设置页新增「能力开关」Tab，替换原「工作模式」Tab，侧边栏不再需要模式感知折叠（管理 API 始终可用）。

**Tech Stack:** Go (net/http ServeMux), Vue 3 SFC, Vite

---

## 文件结构总览

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/config/config.go` | 修改 | 给 `ResponseProxyConfig` / `AnthropicProxyConfig` 加 `Enabled` 字段 |
| `internal/config/config_loader.go` | 修改 | 从 YAML 解析 `enabled` 字段 |
| `internal/service/app/app.go` | 修改 | 移除 `switch mode` 分支，始终调用 `runTransform`；在 Server Config 中注入 proxy 配置 |
| `internal/service/server/server.go` | 修改 | `Server.Config` 加可选 `OpenAIProxy` / `AnthropicProxy` handler；`New()` 中条件挂载 |
| `internal/service/api/router.go` | 修改 | 新增 `GET/PUT /capabilities/proxy` 路由 |
| `internal/service/api/settings.go` | 修改 | 新增 capabilities 读写 handler |
| `internal/config/domain_server.go` | 修改 | `ServerConfig` 加 `OpenAIProxyHandler` / `AnthropicProxyHandler` 字段 |
| `config.example.yml` | 修改 | `proxy` 子节点加 `enabled` 字段 |
| `gui-design/src/api/index.js` | 修改 | 新增 capabilities API 函数 |
| `gui-design/src/views/Settings.vue` | 修改 | 「工作模式」Tab 改为「能力开关」Tab + 开关控件 |
| `gui-design/src/App.vue` | 修改 | 移除 Capture 模式折叠逻辑，侧边栏恢复完整 |
| `gui-design/src/views/Dashboard.vue` | 修改 | 显示活跃的能力状态卡片 |
| `internal/config/convert.go` | 可能修改 | 格式转换适配新字段 |
| `internal/service/store/sqlite_store.go` | 可能修改 | 如果 applySetting 需要处理新字段 |

---

## Chunk 1: 配置层 — 添加 proxy enabled 字段

### Task 1: Config 结构体添加 `Enabled` 字段

**Files:**
- Modify: `internal/config/config.go:214-223`

- [ ] **Step 1: 添加字段**

在 `ResponseProxyConfig` 和 `AnthropicProxyConfig` 中加 `Enabled bool` 字段：

```go
type ResponseProxyConfig struct {
    Enabled         bool   `yaml:"enabled"`
    Model           string
    ProviderBaseURL string `yaml:"base_url"`
    ProviderAPIKey  string `yaml:"api_key"`
}

type AnthropicProxyConfig struct {
    Enabled         bool   `yaml:"enabled"`
    Model           string
    ProviderBaseURL string `yaml:"base_url"`
    ProviderAPIKey  string `yaml:"api_key"`
    ProviderVersion string `yaml:"version"`
}
```

- [ ] **Step 2: 添加辅助方法**

在 `config.go` 中添加便利判断方法：

```go
func (cfg Config) HasOpenAIProxy() bool {
    return cfg.ResponseProxy.Enabled && cfg.ResponseProxy.ProviderBaseURL != ""
}

func (cfg Config) HasAnthropicProxy() bool {
    return cfg.AnthropicProxy.Enabled && cfg.AnthropicProxy.ProviderBaseURL != ""
}
```

- [ ] **Step 3: 更新示例配置**

在 `config.example.yml` 的 proxy 段加 `enabled` 字段：

```yaml
proxy:
  response:
    enabled: false
    base_url: "https://api.openai.com"
    api_key: "replace-with-real-openai-responses-api-key"
    model: "gpt-5.4"
  anthropic:
    enabled: false
    base_url: "https://provider.example.com"
    api_key: "replace-with-real-anthropic-compatible-api-key"
    version: "2023-06-01"
```

- [ ] **Step 4: 同步 config.yml**

更新项目中的 `config.yml` 同样添加 `enabled: false`。

- [ ] **Step 5: 提交**

```bash
git add internal/config/config.go config.example.yml config.yml
git commit -m "feat: add Enabled field to proxy configs for capability toggle"
```

---

## Chunk 2: 服务器层 — 合并启动路径 + 条件挂载 Proxy

### Task 2: Server Config 注入 Proxy Handler

**Files:**
- Modify: `internal/config/domain_server.go:1-12`
- Modify: `internal/service/server/server.go:30-52`

- [ ] **Step 1: ServerConfig 加 handler 字段**

```go
// internal/config/domain_server.go
type ServerConfig struct {
    Addr                string
    AuthToken           string
    Mode                string
    MaxSessions         int
    SessionTTL          string
    OpenAIProxyHandler  http.Handler   // nil = not enabled
    AnthropicProxyHandler http.Handler // nil = not enabled
}
```

需要添加 `import "net/http"`。

- [ ] **Step 2: Server Config struct 加 handler 字段**

```go
// internal/service/server/server.go
type Config struct {
    // ... existing fields ...
    OpenAIProxyHandler   http.Handler
    AnthropicProxyHandler http.Handler
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/config/domain_server.go internal/service/server/server.go
git commit -m "feat: add proxy handler slots to server config"
```

### Task 3: Server.New() 条件挂载 Proxy 路由

**Files:**
- Modify: `internal/service/server/server.go:96-116`

- [ ] **Step 1: 在 New() 中添加条件路由**

找到 `registerPluginRoutes()` 调用后，`return s` 前，插入：

```go
    s.registerPluginRoutes()

    // 可选：透明代理端点
    if cfg.OpenAIProxyHandler != nil {
        s.mux.Handle("/v1/openai/responses", cfg.OpenAIProxyHandler)
        slog.Info("OpenAI 透明代理已启用", "path", "/v1/openai/responses")
    }
    if cfg.AnthropicProxyHandler != nil {
        s.mux.Handle("/v1/anthropic/messages", cfg.AnthropicProxyHandler)
        s.mux.Handle("/anthropic/v1/messages", cfg.AnthropicProxyHandler)
        slog.Info("Anthropic 透明代理已启用", "path", "/v1/anthropic/messages")
    }

    if cfg.Runtime != nil && cfg.Store != nil {
```

- [ ] **Step 2: 提交**

```bash
git add internal/service/server/server.go
git commit -m "feat: conditionally mount proxy endpoints in transform server"
```

### Task 4: app.go 移除 mode switch，统一 runTransform

**Files:**
- Modify: `internal/service/app/app.go:44-56`

- [ ] **Step 1: 重构 RunServer**

保持 `RunServer` 签名不变，内部不再 switch，始终调用 `runTransform`：

```go
func RunServer(ctx context.Context, cfg config.Config, errors io.Writer) error {
    slog.Info("启动服务器", "mode", "Transform", "addr", cfg.Addr)
    return runTransform(ctx, cfg, errors)
}
```

旧的 `runCaptureResponse` 和 `runCaptureAnthropic` 函数保留（可能被第三方引用），但不再被内部调用。加上废弃注释。

- [ ] **Step 2: 在 runTransform 中注入 proxy handler**

找到 `server.New(server.Config{...})` 调用处，添加：

```go
    var openaiProxyHandler, anthropicProxyHandler http.Handler

    if cfg.HasOpenAIProxy() {
        p, err := proxy.NewResponse(proxy.ResponseConfig{
            UpstreamBaseURL: cfg.ResponseProxy.ProviderBaseURL,
            APIKey:          cfg.ResponseProxy.ProviderAPIKey,
            Tracer:          tracer,
            TraceErrors:     errors,
        })
        if err != nil {
            slog.Warn("OpenAI 代理初始化失败，已禁用", "error", err)
        } else {
            openaiProxyHandler = p
            slog.Info("OpenAI 透明代理已就绪", "upstream", cfg.ResponseProxy.ProviderBaseURL)
        }
    }

    if cfg.HasAnthropicProxy() {
        p, err := proxy.NewAnthropic(proxy.AnthropicConfig{
            UpstreamBaseURL: cfg.AnthropicProxy.ProviderBaseURL,
            APIKey:          cfg.AnthropicProxy.ProviderAPIKey,
            Version:         cfg.AnthropicProxy.ProviderVersion,
            Tracer:          tracer,
            TraceErrors:     errors,
        })
        if err != nil {
            slog.Warn("Anthropic 代理初始化失败，已禁用", "error", err)
        } else {
            anthropicProxyHandler = p
            slog.Info("Anthropic 透明代理已就绪", "upstream", cfg.AnthropicProxy.ProviderBaseURL)
        }
    }
```

然后在 `server.Config{...}` 中添加：

```go
    handler := server.New(server.Config{
        // ... existing ...
        OpenAIProxyHandler:   openaiProxyHandler,
        AnthropicProxyHandler: anthropicProxyHandler,
    })
```

- [ ] **Step 3: 向后兼容 — 自动启用 proxy for old configs**

在 `RunServer` 入口添加兼容逻辑：如果 `mode` 为 `CaptureResponse` 且 proxy.response 未显式启用，自动启用：

```go
func RunServer(ctx context.Context, cfg config.Config, errors io.Writer) error {
    // 向后兼容：旧 mode 值自动转换为能力开关
    if cfg.Mode == config.ModeCaptureResponse && !cfg.ResponseProxy.Enabled && cfg.ResponseProxy.ProviderBaseURL != "" {
        cfg.ResponseProxy.Enabled = true
        slog.Info("自动启用 OpenAI 代理（来自旧 mode=CaptureResponse 配置）")
    }
    if cfg.Mode == config.ModeCaptureAnthropic && !cfg.AnthropicProxy.Enabled && cfg.AnthropicProxy.ProviderBaseURL != "" {
        cfg.AnthropicProxy.Enabled = true
        slog.Info("自动启用 Anthropic 代理（来自旧 mode=CaptureAnthropic 配置）")
    }

    slog.Info("启动服务器", "addr", cfg.Addr)
    return runTransform(ctx, cfg, errors)
}
```

- [ ] **Step 4: 提交**

```bash
git add internal/service/app/app.go
git commit -m "feat: unify startup path, enable proxy capabilities via toggle"
```

---

## Chunk 3: 管理 API — capabilities 端点

### Task 5: 添加 capabilities API

**Files:**
- Modify: `internal/service/api/router.go:94-99`
- Modify: `internal/service/api/settings.go:65-70`

- [ ] **Step 1: 注册路由**

在 router.go 的 Settings endpoints 区域添加：

```go
    mux.HandleFunc("GET /capabilities", r.handleGetCapabilities)
    mux.HandleFunc("PUT /capabilities/proxy/response", r.handlePutResponseProxyEnabled)
    mux.HandleFunc("PUT /capabilities/proxy/anthropic", r.handlePutAnthropicProxyEnabled)
```

- [ ] **Step 2: 实现 GET /capabilities handler**

在 settings.go 中添加：

```go
// GET /capabilities
func (r *Router) handleGetCapabilities(w http.ResponseWriter, req *http.Request) {
    cfg := r.runtime.Current()
    respondJSON(w, http.StatusOK, map[string]any{
        "transform":     true, // 始终启用
        "proxy_openai":  cfg.Config.HasOpenAIProxy(),
        "proxy_anthropic": cfg.Config.HasAnthropicProxy(),
    })
}
```

- [ ] **Step 3: 实现 PUT /capabilities/proxy/response handler**

```go
// PUT /capabilities/proxy/response
func (r *Router) handlePutResponseProxyEnabled(w http.ResponseWriter, req *http.Request) {
    var body struct {
        Enabled bool `json:"enabled"`
    }
    if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
        respondError(w, http.StatusBadRequest, "invalid_json", "无效的 JSON 请求体")
        return
    }
    // 暂存 proxy.response.enabled 的变更
    enabledJSON, _ := json.Marshal(body.Enabled)
    chID, err := r.store.StageChange(store.ChangeRow{
        Action:    "update",
        Resource:  "setting",
        TargetKey: "proxy.response.enabled",
        After:     string(enabledJSON),
    })
    if err != nil {
        respondError(w, http.StatusInternalServerError, "stage_error", ...)
        return
    }
    respondJSON(w, http.StatusAccepted, map[string]any{
        "change_id": chID,
        "status":    "pending",
        "message":   "代理开关变更已暂存，应用后需重启服务器生效",
    })
}
```

- [ ] **Step 4: 实现 PUT /capabilities/proxy/anthropic handler**

同结构，target 为 `proxy.anthropic.enabled`。

- [ ] **Step 5: 提交**

```bash
git add internal/service/api/router.go internal/service/api/settings.go
git commit -m "feat: add capabilities API for proxy toggles"
```

---

## Chunk 4: 前端 — 能力开关 GUI

### Task 6: API 层添加 capabilities 函数

**Files:**
- Modify: `gui-design/src/api/index.js`

- [ ] **Step 1: 添加 API 函数**

```js
export function getCapabilities()         { return request('/capabilities') }
export function updateResponseProxy(enabled) {
  return request('/capabilities/proxy/response', { method: 'PUT', body: JSON.stringify({ enabled }) })
}
export function updateAnthropicProxy(enabled) {
  return request('/capabilities/proxy/anthropic', { method: 'PUT', body: JSON.stringify({ enabled }) })
}
```

- [ ] **Step 2: 提交**

```bash
git add gui-design/src/api/index.js
git commit -m "feat: add capabilities API client functions"
```

### Task 7: Settings 页 — 工作模式 Tab 改为能力开关

**Files:**
- Modify: `gui-design/src/views/Settings.vue`

- [ ] **Step 1: 替换 Tab 定义**

```js
const tabs = [
  { key: 'capabilities', label: '能力开关' },
  { key: 'defaults', label: '默认参数' },
  { key: 'websearch', label: 'Web Search' },
  { key: 'extensions', label: '扩展管理' },
]
```

- [ ] **Step 2: 添加能力开关 Tab 内容**

替换原有的 mode Tab 模板内容为：

```html
<!-- Tab: Capabilities -->
<div v-if="activeTab === 'capabilities'" class="card" style="max-width: 600px;">
  <div class="section-title">服务器能力</div>
  <p class="text-secondary" style="font-size:13px; margin-bottom:20px;">
    控制服务器提供的对外 API 通道。开关变更需重启生效。
  </p>

  <!-- Transform (always on) -->
  <div class="flex items-center justify-between mb-4" style="padding:12px; background:var(--bg-primary); border-radius:var(--border-radius-sm);">
    <div>
      <strong>🔀 协议转换引擎</strong>
      <div class="text-secondary" style="font-size:12px;">/v1/responses — 路由 + 协议转换</div>
    </div>
    <span class="tag tag-primary">始终启用</span>
  </div>

  <!-- OpenAI Proxy -->
  <div class="flex items-center justify-between mb-4" style="padding:12px; background:var(--bg-primary); border-radius:var(--border-radius-sm);">
    <div>
      <strong>🔗 OpenAI 透明代理</strong>
      <div class="text-secondary" style="font-size:12px;">/v1/openai/responses — 透传至 OpenAI</div>
    </div>
    <label class="toggle">
      <input type="checkbox" v-model="capsForm.proxy_openai" @change="saveCap('proxy_openai')" />
      <span class="toggle-slider"></span>
    </label>
  </div>

  <!-- Anthropic Proxy -->
  <div class="flex items-center justify-between mb-4" style="padding:12px; background:var(--bg-primary); border-radius:var(--border-radius-sm);">
    <div>
      <strong>🔗 Anthropic 透明代理</strong>
      <div class="text-secondary" style="font-size:12px;">/v1/anthropic/messages — 透传至 Anthropic</div>
    </div>
    <label class="toggle">
      <input type="checkbox" v-model="capsForm.proxy_anthropic" @change="saveCap('proxy_anthropic')" />
      <span class="toggle-slider"></span>
    </label>
  </div>

  <div v-if="capsChanged" class="card mt-4" style="border-color: var(--color-warning); background: var(--color-warning-bg);">
    <p style="font-size:13px; color:var(--color-warning);">
      ⚠️ 能力开关变更通过暂存系统生效，应用后需<strong>重启服务器</strong>。
    </p>
  </div>
</div>
```

- [ ] **Step 3: 添加组件逻辑**

```js
const capsForm = ref({ proxy_openai: false, proxy_anthropic: false })
const capsChanged = ref(false)

async function saveCap(key) {
  try {
    const enabled = capsForm.value[key]
    if (key === 'proxy_openai') await updateResponseProxy(enabled)
    else await updateAnthropicProxy(enabled)
    capsChanged.value = true
    showPostSave()
    showToast('能力开关变更已暂存', 'success')
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  }
}
```

在 fetchData 中加载：

```js
const capRes = await getCapabilities()
capsForm.value = {
  proxy_openai: capRes.proxy_openai,
  proxy_anthropic: capRes.proxy_anthropic,
}
```

- [ ] **Step 4: 提交**

```bash
git add gui-design/src/views/Settings.vue
git commit -m "feat: replace mode selector with capability toggles"
```

### Task 8: App.vue — 移除 Capture 模式折叠

**Files:**
- Modify: `gui-design/src/App.vue`

- [ ] **Step 1: 移除 isCaptureMode 计算属性和条件折叠**

```diff
- const isCaptureMode = computed(() => ...)
- const navItems = computed(() => {
-   if (isCaptureMode.value) return allNavItems.filter(n => !n.mgmt)
-   return allNavItems
- })
+ const navItems = allNavItems  // 始终显示全部导航
```

- [ ] **Step 2: 移除 Capture mode banner（footer 中的橙色提示）**

- [ ] **Step 3: 提交**

```bash
git add gui-design/src/App.vue
git commit -m "refactor: remove capture mode sidebar collapse"
```

### Task 9: Dashboard — 显示能力状态

**Files:**
- Modify: `gui-design/src/views/Dashboard.vue`

- [ ] **Step 1: 在 Dashboard 状态卡片区添加能力状态行**

新增一行显示当前启用的代理通道：

```html
<div class="card-grid card-grid-3 mb-4">
  <div class="card">
    <div class="card-header"><span class="card-title">协议转换</span></div>
    <span class="tag tag-primary">已启用</span>
  </div>
  <div class="card">
    <div class="card-header"><span class="card-title">OpenAI 代理</span></div>
    <span class="tag" :class="caps.proxy_openai ? 'tag-success' : 'tag'">{{ caps.proxy_openai ? '已启用' : '已禁用' }}</span>
  </div>
  <div class="card">
    <div class="card-header"><span class="card-title">Anthropic 代理</span></div>
    <span class="tag" :class="caps.proxy_anthropic ? 'tag-success' : 'tag'">{{ caps.proxy_anthropic ? '已启用' : '已禁用' }}</span>
  </div>
</div>
```

- [ ] **Step 2: 加载能力数据**

在 Dashboard 的 fetchData 中添加 `getCapabilities()` 调用。

- [ ] **Step 3: 提交**

```bash
git add gui-design/src/views/Dashboard.vue
git commit -m "feat: show proxy capability status on dashboard"
```

---

## Chunk 5: 集成测试 & 清理

### Task 10: 端到端验证

- [ ] **Step 1: 编译检查**

```bash
go build ./...
```
预期：编译通过，无错误。

- [ ] **Step 2: 启动服务器**

```bash
go run ./cmd/moonbridge -config config.yml
```
预期日志中显示：
```
level=INFO msg="HTTP 服务器监听中" addr=127.0.0.1:38440
```

- [ ] **Step 3: 验证基础功能**

```bash
curl -s http://127.0.0.1:38440/api/v1/status
curl -s http://127.0.0.1:38440/v1/models
curl -s http://127.0.0.1:38440/api/v1/capabilities
```
预期：status 返回 `{"mode":"Transform",...}`，models 正常，capabilities 返回 `{"transform":true,"proxy_openai":false,"proxy_anthropic":false}`。

- [ ] **Step 4: 启用 OpenAI 代理并验证**

在 config.yml 中设置 `proxy.response.enabled: true`，重启，然后：
```bash
curl -s -X POST http://127.0.0.1:38440/v1/openai/responses -H 'Content-Type: application/json' -d '{"model":"gpt-5.4","input":"hi"}'
```
预期：返回 OpenAI 响应（如果 API key 无效则返回 401，但端点存在）。

- [ ] **Step 5: 验证 GUI**

```bash
cd gui-design && npm run build
```
预期：构建无错误。启动 dev server 验证设置页的能力开关是否加载并正确显示。

- [ ] **Step 6: 清理 — 标记废弃函数**

在 `internal/service/app/app.go` 中为 `runCaptureResponse` 和 `runCaptureAnthropic` 添加：

```go
// Deprecated: runCaptureResponse is no longer used. Proxy capabilities
// are now enabled via config proxy.response.enabled and mounted alongside
// the transform server. Kept for backward compatibility.
func runCaptureResponse(...) error { ... }
```

- [ ] **Step 7: 最终提交**

```bash
git add -A
git commit -m "chore: final integration testing and cleanup"
```

---

## 回滚方案

如果出现严重问题：`config.yml` 中手动设置 `proxy.response.enabled: false` 和 `proxy.anthropic.enabled: false` 即可回退到纯 Transform 行为（与当前模式一致）。

---

> 计划版本：v1.0 | 总预计工时：2-3 hours | 依赖：Go 1.25+, Node 20+, Vue 3
