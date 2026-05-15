<template>
  <div>
    <!-- Loading -->
    <div v-if="loading" class="card-grid card-grid-4">
      <div v-for="i in 4" :key="i" class="card"><div class="skeleton skeleton-card"></div></div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="card" style="border-color: var(--color-danger);">
      <p style="color: var(--color-danger);">加载失败: {{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <!-- Content -->
    <template v-else>
      <!-- Top stat cards -->
      <div class="card-grid card-grid-4 mb-4">
        <div class="card">
          <div class="card-header"><span class="card-title">运行模式</span></div>
          <span class="tag" :class="modeTagClass">{{ status.mode }}</span>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">Provider</span></div>
          <div class="card-value">{{ status.provider_count }}</div>
          <div class="text-secondary" style="font-size: 12px; margin-top: 4px;">
            健康 {{ healthyCount }}/{{ status.provider_count }}
          </div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">路由规则</span></div>
          <div class="card-value">{{ status.route_count }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">监听地址</span></div>
          <div class="text-mono" style="font-size: 14px; font-weight: 500;">{{ status.addr }}</div>
        </div>
      </div>

      <!-- Stats cards -->
      <div class="card-grid card-grid-4 mb-4">
        <div class="card">
          <div class="card-header"><span class="card-title">总请求数</span></div>
          <div class="card-value">{{ fmtNum(summary.requests) }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">Token 消耗</span></div>
          <div class="card-value" style="font-size: 22px;">{{ fmtTokens(summary.input_tokens) }}</div>
          <div class="text-secondary" style="font-size: 12px;">输出 {{ fmtTokens(summary.output_tokens) }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">缓存命中率</span></div>
          <div class="card-value">{{ summary.cache_hit_rate != null ? (summary.cache_hit_rate * 100).toFixed(1) + '%' : '—' }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">累计费用</span></div>
          <div class="card-value" style="font-size: 24px;">¥{{ summary.total_cost != null ? summary.total_cost.toFixed(4) : '0.0000' }}</div>
        </div>
      </div>

      <!-- Capability status -->
      <div class="card-grid card-grid-3 mb-4">
        <div class="card">
          <div class="card-header"><span class="card-title">🔀 协议转换</span></div>
          <div class="flex items-center gap-2" style="padding: 8px 0;">
            <span class="status-dot online"></span>
            <span class="text-primary" style="color:var(--text-primary); font-weight:600;">已启用</span>
          </div>
          <div class="text-secondary" style="font-size:12px;">/v1/responses</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">🔗 OpenAI 代理</span></div>
          <div class="flex items-center gap-2" style="padding: 8px 0;">
            <span class="status-dot" :class="capState.proxy_openai ? 'online' : 'offline'"></span>
            <span :style="{fontWeight:600, color: capState.proxy_openai ? 'var(--color-success)' : 'var(--text-muted)'}">
              {{ capState.proxy_openai ? '已启用' : '已禁用' }}
            </span>
          </div>
          <div class="text-secondary" style="font-size:12px;">/v1/openai/responses</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">🔗 Anthropic 代理</span></div>
          <div class="flex items-center gap-2" style="padding: 8px 0;">
            <span class="status-dot" :class="capState.proxy_anthropic ? 'online' : 'offline'"></span>
            <span :style="{fontWeight:600, color: capState.proxy_anthropic ? 'var(--color-success)' : 'var(--text-muted)'}">
              {{ capState.proxy_anthropic ? '已启用' : '已禁用' }}
            </span>
          </div>
          <div class="text-secondary" style="font-size:12px;">/v1/anthropic/messages</div>
        </div>
      </div>

      <!-- Bottom two columns -->
      <div class="card-grid card-grid-2">
        <!-- Provider health -->
        <div class="card">
          <div class="card-header">
            <span class="card-title">Provider 健康状态</span>
          </div>
          <table class="data-table" v-if="providers.length">
            <thead>
              <tr>
                <th>名称</th><th>协议</th><th>Base URL</th><th>Offers</th><th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in providers" :key="p.key">
                <td><strong>{{ p.key }}</strong></td>
                <td><span class="tag" :class="protocolClass(p.protocol)">{{ p.protocol }}</span></td>
                <td class="mono truncate" style="max-width: 200px;">{{ p.base_url }}</td>
                <td>{{ p.offer_count }}</td>
                <td><span class="status-dot" :class="healthClass(p.health_status)" :title="p.health_status"></span></td>
              </tr>
            </tbody>
          </table>
          <div v-else class="text-secondary" style="padding: 20px 0; text-align: center;">暂无 Provider</div>
        </div>

        <!-- Recent sessions -->
        <div class="card">
          <div class="card-header">
            <span class="card-title">最近会话</span>
          </div>
          <table class="data-table" v-if="sessions.length">
            <thead>
              <tr><th>会话标识</th><th>模型</th><th>最后活跃</th></tr>
            </thead>
            <tbody>
              <tr v-for="s in sessions.slice(0, 8)" :key="s.key">
                <td class="mono">{{ s.key }}</td>
                <td><span class="tag tag-primary">{{ s.model || '—' }}</span></td>
                <td class="text-secondary">{{ timeAgo(s.last_used) }}</td>
              </tr>
            </tbody>
          </table>
          <div v-else class="text-secondary" style="padding: 20px 0; text-align: center;">暂无活跃会话</div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, inject } from 'vue'
import {
  getStatus, getStatsSummary, getStatusProviders, getSessions, getCapabilities,
} from '../api/index.js'

const showToast = inject('showToast')

const loading = ref(true)
const error = ref(null)
const status = ref({ mode: '—', provider_count: 0, route_count: 0, addr: '—' })
const summary = ref({})
const providers = ref([])
const sessions = ref([])
const capState = ref({ proxy_openai: false, proxy_anthropic: false })

let timer = null

function healthClass(h) {
  if (h === 'healthy' || h === 'online') return 'online'
  if (h === 'degraded') return 'degraded'
  if (h === 'offline' || h === 'error') return 'offline'
  return 'unknown'
}

const healthyCount = ref(0)

function protocolClass(proto) {
  const m = {
    'anthropic': 'tag-proto-anthropic',
    'openai-response': 'tag-proto-openai-response',
    'google-genai': 'tag-proto-google-genai',
    'openai-chat': 'tag-proto-openai-chat',
  }
  return m[proto] || 'tag-primary'
}

const modeTagClass = { 'Transform': 'tag-primary', 'CaptureAnthropic': 'tag-success', 'CaptureResponse': 'tag-warning' }

function fmtNum(n) {
  if (n == null) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function fmtTokens(n) {
  if (n == null) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(2) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function timeAgo(ts) {
  if (!ts) return '—'
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours} 小时前`
  return `${Math.floor(hours / 24)} 天前`
}

async function fetchData() {
  try {
    const [st, sm, pr, ss, caps] = await Promise.all([
      getStatus(), getStatsSummary(), getStatusProviders(), getSessions(), getCapabilities(),
    ])
    status.value = st
    summary.value = sm
    providers.value = pr || []
    sessions.value = ss || []
    healthyCount.value = (pr || []).filter(p => p.health_status === 'healthy' || p.health_status === 'online').length
    capState.value = caps || { proxy_openai: false, proxy_anthropic: false }
    error.value = null
  } catch (e) {
    error.value = e.message
    showToast('仪表盘数据加载失败', 'error')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 10000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>
