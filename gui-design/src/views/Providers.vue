<template>
  <div>
    <!-- Toolbar -->
    <div class="flex items-center justify-between mb-4" style="gap: 12px;">
      <div class="flex items-center gap-3">
        <input class="form-input" style="width: 220px;" v-model="search" placeholder="搜索 provider key..." @input="doSearch" />
        <select class="form-select" style="width: 180px;" v-model="protocolFilter" @change="fetchData">
          <option value="">全部协议</option>
          <option value="anthropic">Anthropic</option>
          <option value="openai-response">OpenAI Response</option>
          <option value="google-genai">Google GenAI</option>
          <option value="openai-chat">OpenAI Chat</option>
        </select>
      </div>
      <button class="btn btn-primary" @click="openCreate">＋ 添加 Provider</button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="card-grid card-grid-3">
      <div v-for="i in 6" :key="i" class="card"><div class="skeleton" style="height:120px;"></div></div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="card" style="border-color: var(--color-danger);">
      <p style="color: var(--color-danger);">{{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <!-- Empty -->
    <div v-else-if="!filteredProviders.length" class="empty-state">
      <div class="empty-icon">🔌</div>
      <h3>暂无 Provider</h3>
      <p>添加第一个上游 LLM Provider 开始使用 Moon Bridge</p>
      <button class="btn btn-primary" @click="openCreate">添加 Provider</button>
    </div>

    <!-- Provider Cards -->
    <div v-else class="card-grid card-grid-3">
      <div v-for="p in filteredProviders" :key="p.key" class="card" style="display:flex; flex-direction:column;">
        <!-- Header -->
        <div class="flex items-center justify-between" style="margin-bottom: 12px;">
          <div class="flex items-center gap-2">
            <strong>{{ p.key }}</strong>
            <span class="tag" :class="protocolClass(p.protocol)">{{ p.protocol }}</span>
          </div>
          <span class="status-dot" :class="healthClass(p.health_status)"></span>
        </div>

        <!-- Body -->
        <div style="flex:1; font-size: 12.5px;">
          <div class="text-secondary text-mono truncate" style="margin-bottom: 6px;" :title="p.base_url">{{ p.base_url }}</div>
          <div class="text-secondary">Offers: <strong class="text-primary" style="color:var(--text-primary);">{{ p.offer_count }}</strong></div>
        </div>

        <!-- Footer -->
        <div class="flex items-center gap-2" style="margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--border-color);">
          <button class="btn btn-ghost btn-sm" @click="testConnection(p.key)" :disabled="testingKey === p.key">
            {{ testingKey === p.key ? '测试中...' : '测试连接' }}
          </button>
          <button class="btn btn-ghost btn-sm" @click="openEdit(p)">编辑</button>
          <button class="btn btn-ghost btn-sm" style="color: var(--color-danger);" @click="confirmDelete(p)">删除</button>
        </div>
      </div>
    </div>

    <!-- Test result toast is handled by showToast -->

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <h2 class="modal-title">{{ editingKey ? '编辑 Provider' : '添加 Provider' }}</h2>

        <div class="form-group">
          <label class="form-label">Key *</label>
          <input class="form-input" v-model="form.key" :disabled="!!editingKey" placeholder="如 anthropic-main" />
        </div>

        <div class="form-group">
          <label class="form-label">协议 *</label>
          <select class="form-select" v-model="form.protocol">
            <option value="anthropic">Anthropic Messages</option>
            <option value="openai-response">OpenAI Responses</option>
            <option value="google-genai">Google GenAI</option>
            <option value="openai-chat">OpenAI Chat</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">Base URL *</label>
          <input class="form-input" v-model="form.base_url" placeholder="https://api.anthropic.com" />
        </div>

        <div class="form-group">
          <label class="form-label">API Key *</label>
          <div style="position: relative;">
            <input class="form-input" :type="showKey ? 'text' : 'password'" v-model="form.api_key" :placeholder="editingKey ? '留空则不修改' : 'sk-...'" />
            <button class="btn btn-icon btn-ghost" style="position: absolute; right: 4px; top: 2px;" @click="showKey = !showKey">{{ showKey ? '🙈' : '👁' }}</button>
          </div>
        </div>

        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Version</label>
            <input class="form-input" v-model="form.version" placeholder="2023-06-01" />
          </div>
          <div class="form-group">
            <label class="form-label">User Agent</label>
            <input class="form-input" v-model="form.user_agent" placeholder="moonbridge/1.0" />
          </div>
        </div>

        <div v-if="form.protocol === 'google-genai'" class="card" style="background:var(--bg-primary); margin-bottom:16px;">
          <div class="text-secondary" style="font-size:12px;">
            ⚠️ Google GenAI 专有字段（project / location / api_version）暂不通过此表单管理。<br/>
            请至 <router-link to="/config">配置管理 → YAML 编辑器</router-link> 设置。
          </div>
        </div>

        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeModal">取消</button>
          <button class="btn btn-primary" @click="saveProvider" :disabled="saving">
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirm -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget=null">
      <div class="modal-content" style="max-width: 400px;">
        <h2 class="modal-title">确认删除</h2>
        <p>确定要删除 Provider <strong>{{ deleteTarget.key }}</strong> 吗？</p>
        <p class="text-secondary mt-2" style="font-size: 12px;">此操作将暂存为待处理变更，需在变更管理页应用后生效。</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="deleteTarget=null">取消</button>
          <button class="btn btn-danger" @click="doDelete">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, inject } from 'vue'
import {
  listProviders, createProvider, updateProvider, deleteProvider, testProvider,
} from '../api/index.js'
import { usePostSave } from '../composables/useToast.js'

const showToast = inject('showToast')
const showPostSave = usePostSave()

const loading = ref(true)
const error = ref(null)
const providers = ref([])
const search = ref('')
const protocolFilter = ref('')
const testingKey = ref('')

// Modal state
const showModal = ref(false)
const editingKey = ref('')
const saving = ref(false)
const showKey = ref(false)
const form = ref({ key: '', protocol: 'anthropic', base_url: '', api_key: '', version: '', user_agent: '' })
const deleteTarget = ref(null)

const filteredProviders = computed(() => {
  let list = providers.value
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(p => p.key.toLowerCase().includes(q))
  }
  return list
})

function protocolClass(proto) {
  const m = {
    'anthropic': 'tag-proto-anthropic',
    'openai-response': 'tag-proto-openai-response',
    'google-genai': 'tag-proto-google-genai',
    'openai-chat': 'tag-proto-openai-chat',
  }
  return m[proto] || 'tag-primary'
}

function healthClass(h) {
  if (h === 'healthy' || h === 'online') return 'online'
  if (h === 'degraded') return 'degraded'
  if (h === 'offline' || h === 'error') return 'offline'
  return 'unknown'
}

function doSearch() {} // reactive via computed

function openCreate() {
  editingKey.value = ''
  form.value = { key: '', protocol: 'anthropic', base_url: '', api_key: '', version: '', user_agent: '' }
  showKey.value = false
  showModal.value = true
}

function openEdit(p) {
  editingKey.value = p.key
  form.value = {
    key: p.key,
    protocol: p.protocol,
    base_url: p.base_url,
    api_key: '',
    version: '',
    user_agent: '',
  }
  showKey.value = false
  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

async function saveProvider() {
  if (!form.value.key || !form.value.base_url) {
    showToast('Key 和 Base URL 为必填项', 'warning')
    return
  }
  saving.value = true
  try {
    const body = {
      protocol: form.value.protocol,
      base_url: form.value.base_url,
      api_key: form.value.api_key || undefined,
      version: form.value.version || undefined,
      user_agent: form.value.user_agent || undefined,
    }
    await createProvider(form.value.key, body)
    showPostSave()
    showToast(`Provider "${form.value.key}" 已暂存${editingKey.value ? '更新' : '创建'}`, 'success')
    closeModal()
    fetchData()
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally {
    saving.value = false
  }
}

function confirmDelete(p) {
  deleteTarget.value = p
}

async function doDelete() {
  if (!deleteTarget.value) return
  try {
    await deleteProvider(deleteTarget.value.key)
    showPostSave()
    showToast(`Provider "${deleteTarget.value.key}" 删除已暂存`, 'success')
    deleteTarget.value = null
    fetchData()
  } catch (e) {
    showToast(`删除失败: ${e.message}`, 'error')
  }
}

async function testConnection(key) {
  testingKey.value = key
  try {
    const res = await testProvider(key)
    if (res.success) {
      showToast(`✅ 连接成功 (${res.duration})`, 'success')
    } else {
      showToast(`❌ 连接失败: ${res.error}`, 'error')
    }
  } catch (e) {
    showToast(`测试失败: ${e.message}`, 'error')
  } finally {
    testingKey.value = ''
  }
}

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    const res = await listProviders()
    providers.value = res.data || []
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>
