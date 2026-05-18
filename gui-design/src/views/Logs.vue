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
    <div class="card" style="padding:0; flex:1; display:flex; flex-direction:column; min-height: 500px;">
      <!-- Empty state -->
      <div v-if="!filteredLogs.length" class="empty-state" style="flex:1;">
        <div class="empty-icon">📋</div>
        <h3>暂无日志</h3>
        <p>服务器启动后将自动收集日志</p>
      </div>

      <!-- Log entries -->
      <div v-else ref="logContainer" style="flex:1; overflow-y: auto; padding: 12px; background: #0a0a14; font-family: var(--font-mono); font-size: 12px; line-height: 1.8;">
        <div
          v-for="(log, i) in filteredLogs" :key="i"
          style="border-bottom: 1px solid rgba(255,255,255,0.03); cursor: pointer;"
          :style="{ background: log.level === 'ERROR' ? 'rgba(239,83,80,0.08)' : 'transparent' }"
          @click="toggleExpand(i)"
        >
          <div style="display: flex; gap: 12px; padding: 2px 0;">
            <span style="color: #666; flex-shrink: 0;">{{ log.timestamp || '—' }}</span>
            <span
              style="flex-shrink: 0; min-width: 52px; font-weight: 600;"
              :style="{ color: levelColor(log.level) }"
            >{{ log.level }}</span>
            <span style="color: #ccc; word-break: break-all; flex:1; font-family: var(--font-mono);">
              <span v-if="log.message.match(/^(GET|POST|PUT|DELETE|PATCH)\s/)">
                <span style="font-weight:600;"
                  :style="{color: log.message.includes('→ 2') ? '#4caf50' : log.message.includes('→ 4') ? '#ffa726' : log.message.includes('→ 5') ? '#ef5350' : '#888'}"
                >{{ log.message.split(' ')[0] }}</span>
                <span style="color:#42a5f5; margin:0 4px;">{{ log.message.split(' ')[1] }}</span>
                <span style="color:#888;">→</span>
                <span style="font-weight:600; margin-left:4px;"
                  :style="{color: log.message.includes('→ 2') ? '#4caf50' : log.message.includes('→ 4') ? '#ffa726' : log.message.includes('→ 5') ? '#ef5350' : '#888'}"
                >{{ log.message.split('→')[1]?.trim()?.split(' ')[0] }}</span>
              </span>
              <span v-else>{{ log.message }}</span>
            </span>
            <span v-if="log.attrs" style="color: #555; flex-shrink:0;">{{ expandedIndex === i ? '▲' : '▶' }}</span>
          </div>
          <div v-if="expandedIndex === i && log.attrs" style="padding: 6px 12px 6px 72px; background: rgba(255,255,255,0.03); border-top: 1px solid rgba(255,255,255,0.05); color: #888; font-size: 11px; line-height: 1.7; overflow-x: auto;">
            <div v-for="attr in parseAttrs(log.attrs)" :key="attr.key" style="display: flex; gap: 12px;">
              <span style="color: #666; min-width: 80px; flex-shrink: 0;">{{ attr.key }}</span>
              <span style="color: #aaa; word-break: break-word; white-space: pre-wrap; overflow-wrap: break-word;">{{ attr.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Bottom bar -->
    <div class="flex items-center justify-between mt-2">
      <span class="text-secondary" style="font-size: 12px;">共 {{ filteredLogs.length }} 条日志</span>
      <div class="flex items-center gap-2">
        <span class="status-dot online" style="animation: pulse 2s infinite;"></span>
        <span class="text-secondary" style="font-size: 12px;">实时轮询中</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { getLogs } from '../api/index.js'

const logs = ref([])
const search = ref('')
const levelFilter = ref('ALL')
const autoScroll = ref(true)
const logContainer = ref(null)
const expandedIndex = ref(-1)

function toggleExpand(i) {
  expandedIndex.value = expandedIndex.value === i ? -1 : i
}

function extractPath(s) {
  if (!s) return ''
  const m = s.match(/\bpath=(\/\S+)/)
  return m ? m[1] : ''
}

function parseAttrs(s) {
  if (!s) return []
  return s.split(' ').filter(Boolean).map(p => {
    const idx = p.indexOf('=')
    if (idx === -1) return { key: p, value: '' }
    return { key: p.slice(0, idx), value: p.slice(idx + 1) }
  })
}

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

let pollTimer = null

onMounted(() => {
  fetchLogs()
  pollTimer = setInterval(fetchLogs, 2000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

async function fetchLogs() {
  try {
    const data = await getLogs()
    logs.value = data || []
    if (autoScroll.value && logContainer.value) {
      setTimeout(() => {
        const el = logContainer.value
        if (el) el.scrollTop = el.scrollHeight
      }, 50)
    }
  } catch {}
}
</script>
