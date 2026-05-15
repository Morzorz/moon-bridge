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

      </div>

      <!-- Proxy capability status -->
      <div v-if="capStatus.proxy_openai || capStatus.proxy_anthropic" class="card" style="margin: 8px 10px; padding: 8px 10px; background: var(--color-primary-bg); border-color: var(--color-primary);">
        <div style="font-size:11px; color: var(--color-primary); line-height:1.6;">
          🔗 附加通道：
          <span v-if="capStatus.proxy_openai">OpenAI 代理</span>
          <span v-if="capStatus.proxy_openai && capStatus.proxy_anthropic"> / </span>
          <span v-if="capStatus.proxy_anthropic">Anthropic 代理</span>
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


  </div>
</template>

<script setup>
import { ref, computed, onMounted, provide } from 'vue'
import { useRoute } from 'vue-router'
import { getStatus, getCapabilities } from './api/index.js'

const route = useRoute()
const version = ref('')
const serverMode = ref('')
const capStatus = ref({ proxy_openai: false, proxy_anthropic: false })

const allNavItems = [
  { path: '/',          icon: '📊', label: '仪表盘', mgmt: false },
  { path: '/providers', icon: '🔌', label: 'Providers', mgmt: true },
  { path: '/models',    icon: '🧩', label: '模型管理', mgmt: true },
  { path: '/routes',    icon: '🔀', label: '路由管理', mgmt: true },
  { path: '/settings',  icon: '⚙️', label: '设置', mgmt: true },
  { path: '/config',    icon: '📄', label: '配置管理', mgmt: true },
  { path: '/sessions',  icon: '💬', label: '会话', mgmt: true },
  { path: '/stats',     icon: '📈', label: '统计分析', mgmt: true },
  { path: '/logs',      icon: '📋', label: '日志', mgmt: false },
]

const navItems = allNavItems  // 能力开关架构下始终显示全部导航

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

provide('showToast', showToast)
provide('serverMode', serverMode)

onMounted(async () => {
  try {
    const [st, caps] = await Promise.all([getStatus(), getCapabilities()])
    version.value = st.version
    serverMode.value = st.mode
    capStatus.value = caps
  } catch {
    version.value = '?'
    serverMode.value = 'offline'
  }
})
</script>
