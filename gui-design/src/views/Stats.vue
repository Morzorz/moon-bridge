<template>
  <div>
    <div v-if="loading" class="card-grid card-grid-4">
      <div v-for="i in 5" :key="i" class="card"><div class="skeleton skeleton-card"></div></div>
    </div>

    <div v-else-if="error" class="card" style="border-color:var(--color-danger);">
      <p style="color:var(--color-danger);">{{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <div v-else-if="!hasData" class="empty-state">
      <div class="empty-icon">📈</div>
      <h3>暂无统计数据</h3>
      <p>发送请求后统计数据将自动汇总到这里</p>
    </div>

    <template v-else>
      <!-- Overview cards -->
      <div class="card-grid card-grid-4 mb-4">
        <div class="card">
          <div class="card-header"><span class="card-title">总请求数</span></div>
          <div class="card-value">{{ fmtNum(s.totalRequests || s.requests) }}</div>
          <div class="text-secondary" style="font-size:12px; margin-top:4px;">累计</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">Input Tokens</span></div>
          <div class="card-value" style="font-size:22px;">{{ fmtTokens(s.totalInputTokens || s.input_tokens) }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">Output Tokens</span></div>
          <div class="card-value" style="font-size:22px;">{{ fmtTokens(s.totalOutputTokens || s.output_tokens) }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">缓存命中率</span></div>
          <div class="card-value" style="font-size:24px;">
            {{ s.cache_hit_rate != null ? s.cache_hit_rate.toFixed(1) + '%' : '—' }}
          </div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">缓存写入</span></div>
          <div class="card-value" style="font-size:22px;">{{ fmtTokens(s.totalCacheCreation) }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">缓存读取</span></div>
          <div class="card-value" style="font-size:22px;">{{ fmtTokens(s.totalCacheRead) }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">累计费用</span></div>
          <div class="card-value" style="font-size:22px;">¥{{ s.totalCost != null ? s.totalCost.toFixed(4) : '0.0000' }}</div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">运行时长</span></div>
          <div class="card-value" style="font-size:20px;">{{ s.duration || '—' }}</div>
        </div>
      </div>

      <!-- Charts area (placeholder) -->
      <div class="card-grid card-grid-2 mb-4">
        <div class="card">
          <div class="card-header"><span class="card-title">Token 消耗 (按模型)</span></div>
          <div style="height: 200px; display:flex; align-items:center; justify-content:center;" class="text-secondary">
            <div style="text-align:center;">
              <div style="font-size:32px; margin-bottom:8px;">📊</div>
              图表集成后展示<br/>建议使用 Recharts / ECharts
            </div>
          </div>
        </div>
        <div class="card">
          <div class="card-header"><span class="card-title">费用分布</span></div>
          <div style="height: 200px; display:flex; align-items:center; justify-content:center;" class="text-secondary">
            <div style="text-align:center;">
              <div style="font-size:32px; margin-bottom:8px;">🥧</div>
              图表集成后展示<br/>建议使用 Recharts / ECharts
            </div>
          </div>
        </div>
      </div>

      <!-- Per-model table -->
      <div class="card" style="padding:0; overflow:hidden;">
        <table class="data-table">
          <thead>
            <tr>
              <th>模型名</th><th>请求数</th><th>Input Token</th><th>Output Token</th>
              <th>缓存写入</th><th>缓存读取</th><th>费用 (¥)</th><th>平均费用/请求</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(ms, name) in modelStats" :key="name">
              <td class="mono"><strong>{{ name }}</strong></td>
              <td class="mono">{{ ms.requests }}</td>
              <td class="mono">{{ fmtTokens(ms.inputTokens) }}</td>
              <td class="mono">{{ fmtTokens(ms.outputTokens) }}</td>
              <td class="mono">{{ fmtTokens(ms.cacheCreation) }}</td>
              <td class="mono">{{ fmtTokens(ms.cacheRead) }}</td>
              <td class="mono">¥{{ (ms.cost || 0).toFixed(6) }}</td>
              <td class="mono">¥{{ ms.requests ? ((ms.cost || 0) / ms.requests).toFixed(6) : '0' }}</td>
            </tr>
            <!-- Total row -->
            <tr style="background:var(--bg-tertiary); font-weight:600;">
              <td><strong>合计</strong></td>
              <td class="mono">{{ sum('requests') }}</td>
              <td class="mono">{{ fmtTokens(sum('inputTokens')) }}</td>
              <td class="mono">{{ fmtTokens(sum('outputTokens')) }}</td>
              <td class="mono">{{ fmtTokens(sum('cacheCreation')) }}</td>
              <td class="mono">{{ fmtTokens(sum('cacheRead')) }}</td>
              <td class="mono">¥{{ (sum('cost') || 0).toFixed(6) }}</td>
              <td class="mono">—</td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import { getStats } from '../api/index.js'

const showToast = inject('showToast')

const loading = ref(true)
const error = ref(null)
const data = ref({})

const s = computed(() => data.value || {})

const modelStats = computed(() => {
  return s.value.byModel || {}
})

const hasData = computed(() => {
  return (s.value.totalRequests || s.value.requests) > 0
})

function sum(field) {
  let total = 0
  for (const ms of Object.values(modelStats.value)) {
    total += ms[field] ?? 0
  }
  return total
}

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

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    data.value = await getStats() || {}
  } catch (e) {
    error.value = e.message
  } finally { loading.value = false }
}

onMounted(fetchData)
</script>
