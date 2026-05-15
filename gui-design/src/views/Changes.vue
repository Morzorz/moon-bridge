<template>
  <div>
    <!-- Summary bar -->
    <div class="card mb-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <div>
            <div class="card-value" style="font-size: 32px;">{{ changes.length }}</div>
            <div class="text-secondary" style="font-size: 12px;">待处理变更</div>
          </div>
          <div class="flex gap-3 text-secondary" style="font-size: 13px;">
            <span>Create: <strong style="color:var(--color-success);">{{ countByAction('create') }}</strong></span>
            <span>Update: <strong style="color:var(--color-primary);">{{ countByAction('update') }}</strong></span>
            <span>Delete: <strong style="color:var(--color-danger);">{{ countByAction('delete') }}</strong></span>
          </div>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-success" @click="doApply" :disabled="!changes.length || applying">
            {{ applying ? '应用中...' : '✓ 应用全部变更' }}
          </button>
          <button class="btn btn-danger" @click="confirmDiscard = true" :disabled="!changes.length">
            ✕ 丢弃全部变更
          </button>
        </div>
      </div>
    </div>

    <!-- Config not seeded warning -->
    <div v-if="seedError" class="card mb-4" style="border-color: var(--color-warning); background: var(--color-warning-bg);">
      <div class="flex items-center gap-3">
        <span style="font-size: 20px;">⚠️</span>
        <div style="flex:1;">
          <strong style="color: var(--color-warning);">配置未初始化</strong>
          <p class="mt-2" style="font-size: 13px; color: var(--text-primary);">{{ seedError }}</p>
        </div>
        <router-link to="/config" class="btn btn-primary btn-sm">前往配置管理</router-link>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card"><div class="skeleton" style="height: 200px;"></div></div>

    <!-- Error -->
    <div v-else-if="error" class="card" style="border-color: var(--color-danger);">
      <p style="color: var(--color-danger);">{{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <!-- Empty -->
    <div v-else-if="!changes.length" class="empty-state">
      <div class="empty-icon" style="font-size: 64px;">✓</div>
      <h3>没有待处理的变更</h3>
      <p>在 Provider / 模型 / 路由 / 设置 页面做的修改会自动暂存到这里</p>
    </div>

    <!-- Timeline -->
    <div v-else style="position: relative;">
      <!-- Timeline line -->
      <div style="position: absolute; left: 20px; top: 0; bottom: 0; width: 2px; background: var(--border-color);"></div>

      <div v-for="(ch, idx) in changes" :key="ch.id || idx" style="position: relative; padding-left: 52px; margin-bottom: 16px;">
        <!-- Timeline dot -->
        <div
          style="position: absolute; left: 14px; top: 20px; width: 14px; height: 14px; border-radius: 50%; border: 2px solid; z-index: 1;"
          :style="{ borderColor: dotColor(ch.action), background: dotBg(ch.action) }"
        ></div>

        <!-- Card -->
        <div class="card">
          <div class="flex items-center justify-between" style="margin-bottom: 8px;">
            <div class="flex items-center gap-2">
              <span class="tag" :class="actionTagClass(ch.action)">{{ ch.action?.toUpperCase() }}</span>
              <strong>{{ ch.resource }} "{{ ch.target_key }}"</strong>
            </div>
            <div class="text-secondary text-mono" style="font-size: 11px;">{{ ch.created_at ? timeAgo(ch.created_at) : '—' }}</div>
          </div>

          <!-- Diff view -->
          <div v-if="ch.before || ch.after" class="text-mono" style="font-size: 12px; background: var(--bg-primary); border-radius: var(--border-radius-sm); padding: 12px; max-height: 200px; overflow-y: auto;">
            <div v-if="ch.before && ch.action !== 'create'" style="color: var(--color-danger); white-space: pre-wrap;">- {{ formatJSON(ch.before) }}</div>
            <div v-if="ch.after && ch.action !== 'delete'" style="color: var(--color-success); white-space: pre-wrap;">+ {{ formatJSON(ch.after) }}</div>
          </div>

          <!-- Per-item actions -->
          <div class="flex gap-2" style="margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--border-color);">
            <button class="btn btn-success btn-sm" @click="doApplyOne(ch)" :disabled="applyingOne === ch.id">
              {{ applyingOne === ch.id ? '应用中...' : '✓ 应用此变更' }}
            </button>
            <button class="btn btn-ghost btn-sm" style="color: var(--color-danger);" @click="doDiscardOne(ch)">
              丢弃
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Discard Confirmation -->
    <div v-if="confirmDiscard" class="modal-overlay" @click.self="confirmDiscard=false">
      <div class="modal-content" style="max-width: 420px;">
        <h2 class="modal-title">确认丢弃</h2>
        <p style="color: var(--color-danger);">确定要丢弃所有 {{ changes.length }} 项变更吗？</p>
        <p class="text-secondary mt-2" style="font-size: 12px;">此操作不可撤销。</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="confirmDiscard=false">取消</button>
          <button class="btn btn-danger" @click="doDiscard" :disabled="discarding">{{ discarding ? '丢弃中...' : '确认丢弃' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, inject } from 'vue'
import { listChanges, applyChanges, discardChanges, applyChange, discardChange } from '../api/index.js'

const showToast = inject('showToast')

const loading = ref(true)
const error = ref(null)
const changes = ref([])
const seedError = ref('')
const applying = ref(false)
const discarding = ref(false)
const applyingOne = ref(null)
const confirmDiscard = ref(false)
let timer = null

function dotColor(action) {
  if (action === 'create') return 'var(--color-success)'
  if (action === 'delete') return 'var(--color-danger)'
  return 'var(--color-primary)'
}
function dotBg(action) {
  if (action === 'create') return 'var(--color-success-bg)'
  if (action === 'delete') return 'var(--color-danger-bg)'
  return 'var(--color-primary-bg)'
}
function actionTagClass(action) {
  if (action === 'create') return 'tag-success'
  if (action === 'delete') return 'tag-danger'
  return 'tag-primary'
}

function countByAction(action) {
  return changes.value.filter(c => c.action === action).length
}

function formatJSON(val) {
  if (!val) return ''
  try { return JSON.stringify(typeof val === 'string' ? JSON.parse(val) : val, null, 2) }
  catch { return String(val) }
}

function timeAgo(ts) {
  if (!ts) return '—'
  const diff = Date.now() - new Date(ts).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  const h = Math.floor(mins / 60)
  return `${h} 小时前`
}

async function doApply() {
  applying.value = true
  try {
    await applyChanges()
    showToast(`✅ ${changes.value.length} 项变更已应用生效`, 'success')
    changes.value = []
  } catch (e) {
    const msg = e.message || ''
    // 后端可能已将变更写入 DB 但运行时重载失败
    if (msg.includes('config not seeded') || msg.includes('mode is empty')) {
      showToast('⚠️ 配置未初始化: 请先导入完整 config.yml', 'warning')
      // Show a persistent banner in the UI
      seedError.value = '数据库缺少基础配置（mode 等）。请前往 配置管理 → 导入完整 config.yml，然后再次应用变更。'
    } else {
      showToast(`⚠️ 应用异常: ${msg}`, 'warning')
    }
    // 刷新列表以反映 DB 真实状态（变更记录可能已被标记 applied）
    await fetchData()
  } finally { applying.value = false }
}

async function doDiscard() {
  discarding.value = true
  try {
    await discardChanges()
    showToast(`已丢弃所有变更`, 'info')
    changes.value = []
    confirmDiscard.value = false
  } catch (e) {
    showToast(`丢弃失败: ${e.message}`, 'error')
    await fetchData()
  } finally { discarding.value = false }
}

// === Per-item actions ===
async function doApplyOne(ch) {
  applyingOne.value = ch.id
  try {
    await applyChange(ch.id)
    showToast(`✅ 变更已应用: ${ch.resource} "${ch.target_key}"`, 'success')
    await fetchData()
  } catch (e) {
    showToast(`应用失败: ${e.message}`, 'error')
    await fetchData()
  } finally { applyingOne.value = null }
}

async function doDiscardOne(ch) {
  const id = ch.id
  try {
    await discardChange(id)
    showToast(`已丢弃: ${ch.resource} "${ch.target_key}"`, 'info')
    await fetchData()
  } catch (e) {
    showToast(`丢弃失败: ${e.message}`, 'error')
    await fetchData()
  }
}

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    const result = await listChanges()
    changes.value = result || []
    // If we got a non-empty list, the config IS seeded (changes can exist)
    if (changes.value.length > 0) seedError.value = ''
  } catch (e) {
    if (!changes.value.length) error.value = e.message
  } finally { loading.value = false }
}

onMounted(() => {
  fetchData()
  timer = setInterval(fetchData, 8000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>
