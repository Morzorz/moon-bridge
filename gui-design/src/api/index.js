/**
 * Moon Bridge API 客户端
 *
 * 开发模式: Vite proxy 将 /api/* 转发到 http://127.0.0.1:38440
 * Wails 模式: 通过 Go 后端注入 baseURL，或由 Wails runtime 接管
 */

const BASE_URL = import.meta.env.VITE_API_BASE || '/api/v1'

async function request(path, options = {}) {
  const url = `${BASE_URL}${path}`
  const headers = { 'Content-Type': 'application/json', ...options.headers }

  const res = await fetch(url, { ...options, headers })

  if (!res.ok) {
    let body
    try { body = await res.json() } catch { body = null }
    const msg = body?.error?.message || body?.message || `HTTP ${res.status}`
    throw new Error(msg)
  }

  if (res.status === 204) return null
  return res.json()
}

// ===== Status / Version =====
export function getStatus()           { return request('/status') }
export function getVersion()           { return request('/version') }
export function getStatusProviders()   { return request('/status/providers') }

// ===== Sessions =====
export function getSessions()          { return request('/sessions') }

// ===== Stats =====
export async function getStats() {
  const data = await request('/stats')
  // Go struct serializes as PascalCase (no json tags)
  if (!data || data.message) return data
  return normalizeSummary(data)
}
export function getStatsSummary()      { return request('/stats/summary') }

/** Normalize Go PascalCase -> JS snake_case field names */
function normalizeSummary(data) {
  if (!data) return data
  const norm = {
    requests:       data.Requests ?? data.requests,
    input_tokens:   data.InputTokens ?? data.input_tokens,
    output_tokens:  data.OutputTokens ?? data.output_tokens,
    totalCost:      data.TotalCost ?? data.totalCost,
    cache_hit_rate: data.CacheHitRate ?? data.cache_hit_rate,
    totalCacheCreation: data.CacheCreation ?? data.totalCacheCreation,
    totalCacheRead: data.CacheRead ?? data.totalCacheRead,
    duration:       data.Duration ?? data.duration,
  }
  if (data.ByModel) {
    norm.byModel = {}
    for (const [name, ms] of Object.entries(data.ByModel)) {
      norm.byModel[name] = {
        requests:       ms.Requests ?? ms.requests,
        inputTokens:    ms.InputTokens ?? ms.inputTokens,
        outputTokens:   ms.OutputTokens ?? ms.outputTokens,
        cacheCreation:  ms.CacheCreation ?? ms.cacheCreation,
        cacheRead:      ms.CacheRead ?? ms.cacheRead,
        cost:           ms.Cost ?? ms.cost,
        provider_key:   ms.ProviderKey,
      }
    }
  }
  return norm
}

// ===== Providers =====
export function listProviders(params = {}) {
  const q = new URLSearchParams(params).toString()
  return request(`/providers${q ? '?' + q : ''}`)
}
export function getProvider(key)       { return request(`/providers/${encodeURIComponent(key)}`) }
export function createProvider(key, body) {
  return request(`/providers/${encodeURIComponent(key)}`, { method: 'PUT', body: JSON.stringify(body) })
}
export function updateProvider(key, body) {
  return request(`/providers/${encodeURIComponent(key)}`, { method: 'PATCH', body: JSON.stringify(body) })
}
export function deleteProvider(key) {
  return request(`/providers/${encodeURIComponent(key)}`, { method: 'DELETE' })
}
export function testProvider(key) {
  return request(`/providers/${encodeURIComponent(key)}/test`, { method: 'POST' })
}

// ===== Offers =====
export function createOffer(providerKey, body) {
  return request(`/providers/${encodeURIComponent(providerKey)}/offers`, { method: 'POST', body: JSON.stringify(body) })
}
export function updateOffer(providerKey, model, body) {
  return request(`/providers/${encodeURIComponent(providerKey)}/offers/${encodeURIComponent(model)}`, { method: 'PATCH', body: JSON.stringify(body) })
}
export function deleteOffer(providerKey, model) {
  return request(`/providers/${encodeURIComponent(providerKey)}/offers/${encodeURIComponent(model)}`, { method: 'DELETE' })
}

// ===== Models =====
export function listModels(params = {}) {
  const q = new URLSearchParams(params).toString()
  return request(`/models${q ? '?' + q : ''}`)
}
export function getModel(slug)         { return request(`/models/${encodeURIComponent(slug)}`) }
export function createModel(slug, body) {
  return request(`/models/${encodeURIComponent(slug)}`, { method: 'PUT', body: JSON.stringify(body) })
}
export function deleteModel(slug) {
  return request(`/models/${encodeURIComponent(slug)}`, { method: 'DELETE' })
}

// ===== Routes =====
export function listRoutes(params = {}) {
  const q = new URLSearchParams(params).toString()
  return request(`/routes${q ? '?' + q : ''}`)
}
export function getRoute(alias)        { return request(`/routes/${encodeURIComponent(alias)}`) }
export function createRoute(alias, body) {
  return request(`/routes/${encodeURIComponent(alias)}`, { method: 'PUT', body: JSON.stringify(body) })
}
export function deleteRoute(alias) {
  return request(`/routes/${encodeURIComponent(alias)}`, { method: 'DELETE' })
}

// ===== Settings =====
export function getMode()              { return request('/settings/mode') }
export function updateMode(mode)       { return request('/settings/mode', { method: 'PUT', body: JSON.stringify({ mode }) }) }
export function getCapabilities()      { return request('/capabilities') }
export function updateResponseProxy(enabled) {
  return request('/capabilities/proxy/response', { method: 'PUT', body: JSON.stringify({ enabled }) })
}
export function updateAnthropicProxy(enabled) {
  return request('/capabilities/proxy/anthropic', { method: 'PUT', body: JSON.stringify({ enabled }) })
}
export function getDefaults()          { return request('/defaults') }
export function updateDefaults(body) {
  return request('/defaults', { method: 'PUT', body: JSON.stringify(body) })
}
export function getWebSearch()         { return request('/web-search') }
export function updateWebSearch(body) {
  return request('/web-search', { method: 'PUT', body: JSON.stringify(body) })
}
export function listExtensions()       { return request('/extensions') }
export function getExtension(name)     { return request(`/extensions/${encodeURIComponent(name)}`) }
export function updateExtension(name, body) {
  return request(`/extensions/${encodeURIComponent(name)}`, { method: 'PUT', body: JSON.stringify(body) })
}

// ===== Config =====
export function getConfigEffective()   { return request('/config/effective') }
export function exportConfig()         { return request('/config/export') }
export function importConfig(yaml)     { return request('/config/import', { method: 'POST', body: JSON.stringify({ yaml }) }) }
export function validateConfig(yaml)   { return request('/config/validate', { method: 'POST', body: JSON.stringify({ config: yaml }) }) }

// ===== Changes =====
export async function listChanges() {
  const data = await request('/changes')
  if (!Array.isArray(data)) return data
  return data.map(normalizeChangeRow)
}
export function applyChanges()         { return request('/changes/apply', { method: 'POST' }) }
export function discardChanges()       { return request('/changes/discard', { method: 'POST' }) }
export function applyChange(id)        { return request(`/changes/${id}/apply`, { method: 'POST' }) }
export function discardChange(id)      { return request(`/changes/${id}/discard`, { method: 'POST' }) }

/** Normalize Go PascalCase -> JS snake_case for ChangeRow */
function normalizeChangeRow(r) {
  return {
    id:         r.ID ?? r.id,
    batch_id:   r.BatchID ?? r.batch_id,
    action:     r.Action ?? r.action,
    resource:   r.Resource ?? r.resource,
    target_key: r.TargetKey ?? r.target_key,
    before:     r.Before ?? r.before,
    after:      r.After ?? r.after,
    applied:    r.Applied ?? r.applied,
    error:      r.Error ?? r.error,
    revision:   r.Revision ?? r.revision,
    created_at: r.CreatedAt ?? r.created_at,
    applied_at: r.AppliedAt ?? r.applied_at,
  }
}

// ===== Logs =====
export function getLogs()              { return request('/logs') }

// ===== Config Export (returns YAML text, not JSON) =====
export async function exportConfigRaw() {
  const url = `${BASE_URL}/config/export`
  const res = await fetch(url)
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try { const b = await res.json(); msg = b?.error?.message || msg } catch {}
    throw new Error(msg)
  }
  return res.text()
}
