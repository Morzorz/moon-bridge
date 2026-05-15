<template>
  <div>
    <!-- Controls -->
    <div class="card mb-4">
      <div class="flex items-center justify-between" style="flex-wrap: wrap; gap: 10px;">
        <div class="flex items-center gap-2">
          <button
            v-for="lvl in levels" :key="lvl"
            class="btn btn-ghost btn-sm"
            :style="levelFilter === lvl ? { background: levelBg(lvl), color: levelColor(lvl) } : {}"
            @click="levelFilter = lvl"
          >{{ lvl }}</button>
        </div>
        <div class="flex items-center gap-2">
          <input class="form-input" style="width: 200px;" v-model="search" placeholder="搜索日志..." />
          <button class="btn btn-ghost btn-sm" @click="autoScroll = !autoScroll" :style="autoScroll ? {color:'var(--color-primary)'} : {}">
            {{ autoScroll ? '📌 自动滚动' : '📍 已固定' }}
          </button>
          <button class="btn btn-ghost btn-sm" @click="logs = []">清空</button>
        </div>
      </div>
    </div>

    <!-- Log area -->
    <div class="card" style="padding:0; overflow:hidden; flex:1; display:flex; flex-direction:column; min-height: 500px;">
      <!-- Empty state -->
      <div v-if="!filteredLogs.length" class="empty-state" style="flex:1;">
        <div class="empty-icon">📋</div>
        <h3>日志功能即将上线</h3>
        <p>未来将支持实时滚动查看服务器运行日志</p>
      </div>

      <!-- Log entries -->
      <div v-else ref="logContainer" style="flex:1; overflow-y: auto; padding: 12px; background: #0a0a14; font-family: var(--font-mono); font-size: 12px; line-height: 1.8;">
        <div
          v-for="(log, i) in filteredLogs" :key="i"
          style="display: flex; gap: 12px; padding: 2px 0; border-bottom: 1px solid rgba(255,255,255,0.03);"
          :style="{ background: log.level === 'ERROR' ? 'rgba(239,83,80,0.08)' : 'transparent' }"
        >
          <span style="color: #666; flex-shrink: 0;">{{ log.timestamp || '—' }}</span>
          <span
            style="flex-shrink: 0; min-width: 52px; font-weight: 600;"
            :style="{ color: levelColor(log.level) }"
          >{{ log.level }}</span>
          <span style="color: #ccc; word-break: break-all;">{{ log.message }}</span>
        </div>
      </div>
    </div>

    <!-- Bottom bar -->
    <div class="flex items-center justify-between mt-2">
      <span class="text-secondary" style="font-size: 12px;">共 {{ filteredLogs.length }} 条日志</span>
      <div class="flex items-center gap-2">
        <span class="status-dot offline" style="animation: pulse 2s infinite;"></span>
        <span class="text-secondary" style="font-size: 12px;">占位 — 后端日志 API 待实现</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getLogs } from '../api/index.js'

const logs = ref([])
const search = ref('')
const levelFilter = ref('ALL')
const autoScroll = ref(true)
const logContainer = ref(null)

const levels = ['ALL', 'DEBUG', 'INFO', 'WARN', 'ERROR']

const filteredLogs = computed(() => {
  let list = logs.value
  if (levelFilter.value !== 'ALL') {
    list = list.filter(l => l.level === levelFilter.value)
  }
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(l => (l.message || '').toLowerCase().includes(q))
  }
  return list
})

function levelColor(lvl) {
  const m = { DEBUG: '#888', INFO: '#42a5f5', WARN: '#ffa726', ERROR: '#ef5350' }
  return m[lvl] || '#ccc'
}

function levelBg(lvl) {
  const m = { DEBUG: 'rgba(255,255,255,0.05)', INFO: 'rgba(66,165,245,0.12)', WARN: 'rgba(255,167,38,0.12)', ERROR: 'rgba(239,83,80,0.12)' }
  return m[lvl] || 'transparent'
}

onMounted(async () => {
  try {
    logs.value = await getLogs() || []
  } catch {}
})
</script>
