<template>
  <div class="app-layout">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="sidebar-logo">
          <span class="logo-icon">🌉</span>
          Moon Bridge
        </div>
        <div class="sidebar-subtitle">LLM 协议转换与模型路由</div>
      </div>

      <nav class="sidebar-nav">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: $route.path === item.path }"
        >
          <span class="nav-icon">{{ item.icon }}</span>
          {{ item.label }}
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <span>v{{ version || 'dev' }}</span>
        <span>{{ serverMode || '—' }}</span>
        <span v-if="isCaptureMode" class="tag tag-warning" style="margin-top:4px;">代理模式 — 管理受限</span>
      </div>

      <!-- Capture mode banner -->
      <div v-if="isCaptureMode" class="card" style="margin: 8px 10px; padding: 10px; background: var(--color-warning-bg); border-color: var(--color-warning);">
        <div style="font-size:12px; color: var(--color-warning); line-height:1.6;">
          ⚠️ 服务器运行于 <strong>{{ serverMode }}</strong> 模式<br/>
          管理 API 不可用，仅仪表盘和日志可用。<br/>
          <router-link to="/settings" style="font-size:11px;">切换至 Transform 模式</router-link>
        </div>
      </div>
    </aside>

    <main class="main-area">
      <div class="page-header">
        <h1 class="page-title">{{ pageTitle }}</h1>
        <p class="page-subtitle" v-if="pageSubtitle">{{ pageSubtitle }}</p>
      </div>

      <div class="page-content">
        <router-view />
      </div>
    </main>

    <!-- Toast Container -->
    <div class="toast-container">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="toast"
        :class="`toast-${toast.type}`"
      >
        <span>{{ toast.message }}</span>
      </div>
    </div>

    <!-- Post-Save Dialog -->
    <div v-if="postSaveVisible" class="modal-overlay" @click.self="postSaveVisible = false">
      <div class="modal-content" style="max-width: 420px;">
        <div style="text-align: center; margin-bottom: 16px;">
          <div style="font-size: 40px; margin-bottom: 8px;">📋</div>
          <h2 class="modal-title" style="margin:0;">变更已暂存 ✓</h2>
        </div>
        <p style="text-align: center; color: var(--text-secondary); font-size: 14px; line-height: 1.7;">
          修改已记录，但尚未生效。<br/>
          需要前往<strong>变更管理</strong>页面统一应用后才会生效。
        </p>
        <div class="modal-actions" style="flex-direction: column; gap: 8px; border: none;">
          <router-link to="/changes" class="btn btn-primary w-full" style="justify-content:center;" @click="postSaveVisible = false">
            ➜ 前往变更管理并应用
          </router-link>
          <button class="btn btn-ghost w-full" @click="postSaveVisible = false">稍后处理</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, provide } from 'vue'
import { useRoute } from 'vue-router'
import { getStatus } from './api/index.js'

const route = useRoute()
const version = ref('')
const serverMode = ref('')

const allNavItems = [
  { path: '/',          icon: '📊', label: '仪表盘', mgmt: false },
  { path: '/providers', icon: '🔌', label: 'Providers', mgmt: true },
  { path: '/models',    icon: '🧩', label: '模型管理', mgmt: true },
  { path: '/routes',    icon: '🔀', label: '路由管理', mgmt: true },
  { path: '/settings',  icon: '⚙️', label: '设置', mgmt: true },
  { path: '/config',    icon: '📄', label: '配置管理', mgmt: true },
  { path: '/changes',   icon: '🔄', label: '变更管理', mgmt: true },
  { path: '/sessions',  icon: '💬', label: '会话', mgmt: true },
  { path: '/stats',     icon: '📈', label: '统计分析', mgmt: true },
  { path: '/logs',      icon: '📋', label: '日志', mgmt: false },
]

const isCaptureMode = computed(() => serverMode.value === 'CaptureResponse' || serverMode.value === 'CaptureAnthropic')
const navItems = computed(() => {
  // Capture 模式下只显示基础页面（仪表盘 + 日志）
  if (isCaptureMode.value) {
    return allNavItems.filter(n => !n.mgmt)
  }
  return allNavItems
})

const pageTitle = computed(() => route.meta?.title || 'Moon Bridge')
const pageSubtitle = computed(() => {
  const map = {
    'dashboard': '服务器整体运行状态与关键指标',
    'providers': '管理上游 LLM Provider 及连通性测试',
    'models': '模型定义及各 Provider 定价管理',
    'routes': '模型别名到上游 Provider 的路由映射',
    'settings': '默认参数、Web Search 与扩展配置',
    'config': '查看、验证、导入、导出配置',
    'changes': '暂存变更的统一应用与丢弃',
    'sessions': '活跃客户端会话列表',
    'stats': '用量统计可视化',
    'logs': '服务器运行日志',
  }
  return map[route.name] || ''
})

// ===== Toast System =====
const toasts = ref([])
let toastId = 0

function showToast(message, type = 'info', duration = 3500) {
  const id = ++toastId
  toasts.value.push({ id, message, type })
  setTimeout(() => {
    toasts.value = toasts.value.filter(t => t.id !== id)
  }, duration)
}

// ===== Post-Save Dialog =====
const postSaveVisible = ref(false)
function showPostSave() {
  postSaveVisible.value = true
}

provide('showToast', showToast)
provide('showPostSave', showPostSave)
provide('serverMode', serverMode)
provide('isCaptureMode', isCaptureMode)

onMounted(async () => {
  try {
    const st = await getStatus()
    version.value = st.version
    serverMode.value = st.mode
  } catch {
    version.value = '?'
    serverMode.value = 'offline'
  }
})
</script>
