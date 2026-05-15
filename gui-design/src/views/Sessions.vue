<template>
  <div>
    <!-- Auto-refresh indicator -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center gap-2">
        <span>活跃会话总数: <strong>{{ sessions.length }}</strong></span>
        <span class="text-secondary" style="font-size: 12px;">· {{ uniqueModels }} 个不同模型</span>
      </div>
      <div class="flex items-center gap-2">
        <span class="status-dot" :class="autoRefresh ? 'online' : 'offline'" style="animation: pulse 2s infinite;"></span>
        <span class="text-secondary" style="font-size: 12px;">{{ autoRefresh ? '自动刷新中' : '已暂停' }}</span>
        <button class="btn btn-ghost btn-sm" @click="autoRefresh = !autoRefresh">
          {{ autoRefresh ? '暂停' : '恢复' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="card"><div class="skeleton" style="height:200px;"></div></div>
    <div v-else-if="error" class="card" style="border-color:var(--color-danger);">
      <p style="color:var(--color-danger);">{{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <div v-else-if="!sessions.length" class="empty-state">
      <div class="empty-icon">💬</div>
      <h3>当前没有活跃会话</h3>
      <p>发送 API 请求后会话将自动创建</p>
    </div>

    <div v-else class="card" style="padding:0; overflow:hidden;">
      <table class="data-table">
        <thead>
          <tr>
            <th>会话标识</th><th>使用模型</th><th>创建时间</th><th>最后活跃</th><th>存活时长</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="s in sortedSessions" :key="s.key">
            <td class="mono">{{ s.key }}</td>
            <td><span class="tag tag-primary">{{ s.model || '—' }}</span></td>
            <td class="text-secondary">{{ fmtTime(s.created_at) }}</td>
            <td class="text-secondary" :title="s.last_used">{{ timeAgo(s.last_used) }}</td>
            <td class="mono">{{ uptime(s.created_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, inject } from 'vue'
import { getSessions } from '../api/index.js'

const showToast = inject('showToast')

const loading = ref(true)
const error = ref(null)
const sessions = ref([])
const autoRefresh = ref(true)
let timer = null

const uniqueModels = computed(() => {
  const set = new Set(sessions.value.map(s => s.model).filter(Boolean))
  return set.size
})

const sortedSessions = computed(() => {
  return [...sessions.value].sort((a, b) => {
    return new Date(b.last_used || b.created_at) - new Date(a.last_used || a.created_at)
  })
})

function fmtTime(ts) {
  if (!ts) return '—'
  const d = new Date(ts)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function timeAgo(ts) {
  if (!ts) return '—'
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins}m`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return `${h}h ${m}m`
}

function uptime(ts) {
  if (!ts) return '—'
  const diff = Date.now() - new Date(ts).getTime()
  const h = Math.floor(diff / 3600000)
  const m = Math.floor((diff % 3600000) / 60000)
  return `${h}h ${m}m`
}

async function fetchData() {
  loading.value = false
  error.value = null
  try {
    sessions.value = await getSessions() || []
  } catch (e) {
    if (!sessions.value.length) error.value = e.message
  }
}

onMounted(() => {
  fetchData()
  timer = setInterval(() => { if (autoRefresh.value) fetchData() }, 15000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>
