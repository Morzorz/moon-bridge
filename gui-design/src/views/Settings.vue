<template>
  <div>
    <!-- Tabs -->
    <div class="flex gap-2 mb-4" style="border-bottom: 1px solid var(--border-color); padding-bottom: 0;">
      <button
        v-for="tab in tabs" :key="tab.key"
        class="btn btn-ghost btn-sm"
        :class="{ 'tab-active': activeTab === tab.key }"
        @click="activeTab = tab.key"
        style="border-radius: 6px 6px 0 0; border-bottom: 2px solid transparent;"
        :style="activeTab === tab.key ? { borderBottomColor: 'var(--color-primary)', color: 'var(--color-primary)' } : {}"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Tab: Mode -->
    <div v-if="activeTab === 'mode'" class="card" style="max-width: 600px;">
      <div class="section-title">服务器工作模式</div>
      <p class="text-secondary" style="font-size: 13px; margin-bottom: 16px;">
        当前模式：<span class="tag tag-primary">{{ currentMode }}</span>
      </p>

      <div class="form-group">
        <label class="form-label">切换模式</label>
        <select class="form-select" v-model="modeForm.mode" @change="modeChanged = true">
          <option value="Transform">Transform — 协议转换 + 模型路由（完整功能）</option>
          <option value="CaptureResponse">CaptureResponse — OpenAI Responses 透明代理</option>
          <option value="CaptureAnthropic">CaptureAnthropic — Anthropic Messages 透明代理</option>
        </select>
      </div>

      <!-- Mode description card -->
      <div class="card mt-4" style="background: var(--bg-primary);">
        <div style="font-size: 13px; line-height: 1.8;">
          <div v-if="modeForm.mode === 'Transform'" style="color: var(--color-primary);">
            <strong>Transform 模式</strong><br/>
            接收 OpenAI Responses 格式请求<br/>
            按路由定义自动转换协议并发往上游 Provider<br/>
            完整支持：管理 API、Provider/模型/路由管理、协议转换、Web Search、扩展<br/>
            这是功能最完整的模式。
          </div>
          <div v-if="modeForm.mode === 'CaptureResponse'" style="color: var(--color-warning);">
            <strong>CaptureResponse 模式</strong><br/>
            OpenAI Responses 透明代理，不做任何协议转换<br/>
            管理 API 不可用，只能通过 config.yml 配置<br/>
            切换后需重启服务器生效。
          </div>
          <div v-if="modeForm.mode === 'CaptureAnthropic'" style="color: var(--color-warning);">
            <strong>CaptureAnthropic 模式</strong><br/>
            Anthropic Messages 透明代理，不做任何协议转换<br/>
            管理 API 不可用，只能通过 config.yml 配置<br/>
            切换后需重启服务器生效。
          </div>
        </div>
      </div>

      <div v-if="modeChanged && modeForm.mode !== currentMode" class="card mt-4" style="border-color: var(--color-warning); background: var(--color-warning-bg);">
        <p style="font-size: 13px; color: var(--color-warning);">
          ⚠️ 模式切换需要<strong>重启服务器</strong>才能完全生效。<br/>
          变更会暂存到数据库，应用后在下一次启动时生效。
        </p>
      </div>

      <button class="btn btn-primary mt-4" @click="saveMode" :disabled="savingMode || modeForm.mode === currentMode">
        {{ savingMode ? '保存中...' : '保存并暂存变更' }}
      </button>
    </div>

    <!-- Tab: Defaults -->
    <div v-if="activeTab === 'defaults'" class="card" style="max-width: 600px;">
      <div class="form-group">
        <label class="form-label">默认模型</label>
        <select class="form-select" v-model="defaultsForm.model">
          <option value="">无默认值</option>
          <option v-for="opt in defaultModelOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label">默认 Max Tokens</label>
        <input class="form-input" type="number" v-model.number="defaultsForm.max_tokens" placeholder="4096" />
      </div>
      <div class="form-group">
        <label class="form-label">默认 System Prompt</label>
        <textarea class="form-textarea" rows="4" v-model="defaultsForm.system_prompt" placeholder="可选系统提示词"></textarea>
      </div>
      <button class="btn btn-primary" @click="saveDefaults" :disabled="savingDefaults">{{ savingDefaults ? '保存中...' : '保存' }}</button>
      <p class="text-secondary mt-2" style="font-size: 12px;">当客户端请求未指定相应值时使用这些默认值</p>
    </div>

    <!-- Tab: Web Search -->
    <div v-if="activeTab === 'websearch'" class="card" style="max-width: 600px;">
      <div class="form-group">
        <label class="form-label">搜索支持模式</label>
        <select class="form-select" v-model="webForm.support">
          <option value="disabled">disabled — 完全禁用</option>
          <option value="enabled">enabled — 客户端可主动调用</option>
          <option value="auto">auto — 自动注入搜索工具</option>
        </select>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label class="form-label">最大使用次数</label>
          <input class="form-input" type="number" v-model.number="webForm.max_uses" />
        </div>
        <div class="form-group">
          <label class="form-label">最大搜索轮次</label>
          <input class="form-input" type="number" v-model.number="webForm.search_max_rounds" />
        </div>
      </div>
      <div class="form-group">
        <label class="form-label">Tavily API Key</label>
        <div style="position:relative;">
          <input class="form-input" :type="showTavily ? 'text' : 'password'" v-model="webForm.tavily_api_key" placeholder="留空则不修改" />
          <button class="btn btn-icon btn-ghost" style="position:absolute; right:4px; top:2px;" @click="showTavily = !showTavily">{{ showTavily ? '🙈' : '👁' }}</button>
        </div>
      </div>
      <div class="form-group">
        <label class="form-label">Firecrawl API Key</label>
        <div style="position:relative;">
          <input class="form-input" :type="showFirecrawl ? 'text' : 'password'" v-model="webForm.firecrawl_api_key" placeholder="留空则不修改" />
          <button class="btn btn-icon btn-ghost" style="position:absolute; right:4px; top:2px;" @click="showFirecrawl = !showFirecrawl">{{ showFirecrawl ? '🙈' : '👁' }}</button>
        </div>
      </div>
      <button class="btn btn-primary" @click="saveWebSearch" :disabled="savingWeb">{{ savingWeb ? '保存中...' : '保存' }}</button>
    </div>

    <!-- Tab: Extensions -->
    <div v-if="activeTab === 'extensions'" style="max-width: 700px;">
      <div v-if="extensions.length === 0" class="empty-state" style="padding:40px 20px;">
        <div class="empty-icon">🧩</div>
        <h3>暂无扩展</h3>
        <p>当前系统没有注册任何扩展</p>
      </div>

      <div v-for="ext in extensions" :key="ext.name || ext" class="card mb-4">
        <div class="flex items-center justify-between" style="margin-bottom:12px;">
          <h3 class="section-title" style="margin:0;">{{ typeof ext === 'string' ? ext : ext.name }}</h3>
          <span class="tag tag-primary">Extension</span>
        </div>
        <p class="text-secondary" style="font-size:12px;">编辑扩展配置将暂存为 pending change</p>
        <button class="btn btn-ghost btn-sm mt-2" @click="configureExt(ext)">配置</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import {
  getDefaults, updateDefaults,
  getWebSearch, updateWebSearch,
  listExtensions, getMode, updateMode,
} from '../api/index.js'
import { usePostSave } from '../composables/useToast.js'

const showToast = inject('showToast')
const showPostSave = usePostSave()

const tabs = [
  { key: 'mode', label: '工作模式' },
  { key: 'defaults', label: '默认参数' },
  { key: 'websearch', label: 'Web Search' },
  { key: 'extensions', label: '扩展管理' },
]
const activeTab = ref('defaults')

const currentMode = ref('')
const modeChanged = ref(false)
const savingMode = ref(false)
const modeForm = ref({ mode: 'Transform' })
const modelList = ref([])
const defaultModelOptions = ref([])
const extensions = ref([])

// Defaults
const defaultsForm = ref({ model: '', max_tokens: null, system_prompt: '' })
const savingDefaults = ref(false)

// Web Search
const webForm = ref({ support: 'disabled', max_uses: null, tavily_api_key: '', firecrawl_api_key: '', search_max_rounds: null })
const showTavily = ref(false)
const showFirecrawl = ref(false)
const savingWeb = ref(false)

async function saveMode() {
  if (modeForm.value.mode === currentMode.value) return
  savingMode.value = true
  try {
    await updateMode(modeForm.value.mode)
    showPostSave()
    showToast(`Mode 变更已暂存: ${modeForm.value.mode}`, 'success')
    modeChanged.value = false
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally { savingMode.value = false }
}

async function saveDefaults() {
  savingDefaults.value = true
  try {
    const body = {
      model: defaultsForm.value.model || undefined,
      max_tokens: defaultsForm.value.max_tokens || undefined,
      system_prompt: defaultsForm.value.system_prompt || undefined,
    }
    await updateDefaults(body)
    showPostSave()
    showToast('默认参数已暂存，需在变更管理页应用', 'success')
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally { savingDefaults.value = false }
}

async function saveWebSearch() {
  savingWeb.value = true
  try {
    await updateWebSearch({
      support: webForm.value.support,
      max_uses: webForm.value.max_uses || undefined,
      tavily_api_key: webForm.value.tavily_api_key || undefined,
      firecrawl_api_key: webForm.value.firecrawl_api_key || undefined,
      search_max_rounds: webForm.value.search_max_rounds || undefined,
    })
    showPostSave()
    showToast('Web Search 配置已暂存，需在变更管理页应用', 'success')
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally { savingWeb.value = false }
}

function configureExt(ext) {
  showToast(`扩展 "${typeof ext === 'string' ? ext : ext.name}" 配置功能开发中`, 'info')
}

async function fetchData() {
  try {
    const [mo, d, w, e, mRes, rRes] = await Promise.all([
      getMode(), getDefaults(), getWebSearch(), listExtensions(),
      import('../api/index.js').then(m => m.listModels({ limit: 100 })),
      import('../api/index.js').then(m => m.listRoutes({ limit: 100 })),
    ])
    currentMode.value = mo.mode
    modeForm.value = { mode: mo.mode }
    defaultsForm.value = { model: d.model || '', max_tokens: d.max_tokens, system_prompt: d.system_prompt || '' }
    webForm.value = {
      support: w.support || 'disabled',
      max_uses: w.max_uses,
      tavily_api_key: '',
      firecrawl_api_key: '',
      search_max_rounds: w.search_max_rounds,
    }
    extensions.value = e || []
    modelList.value = mRes.data || []
    // Build combined default-model options: route aliases first, then model slugs
    const routes = (rRes.data || []).map(r => ({ label: `${r.alias} （路由）`, value: r.alias }))
    const models = (mRes.data || []).map(m => ({ label: `${m.slug} （模型）`, value: m.slug }))
    defaultModelOptions.value = [...routes, ...models]
  } catch (e) {
    showToast(`加载设置失败: ${e.message}`, 'error')
  }
}

onMounted(fetchData)
</script>
