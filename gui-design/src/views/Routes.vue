<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <div></div>
      <button class="btn btn-primary" @click="openCreate">＋ 添加路由</button>
    </div>

    <div v-if="loading" class="card"><div class="skeleton" style="height:200px;"></div></div>
    <div v-else-if="error" class="card" style="border-color:var(--color-danger);">
      <p style="color:var(--color-danger);">{{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <div v-else-if="!routes.length" class="empty-state">
      <div class="empty-icon">🔀</div>
      <h3>还没有路由规则</h3>
      <p>创建第一条路由将客户端模型名映射到上游 Provider</p>
      <button class="btn btn-primary" @click="openCreate">添加路由</button>
    </div>

    <div v-else class="card" style="padding:0; overflow:hidden;">
      <table class="data-table">
        <thead>
          <tr><th>Alias</th><th>目标模型</th><th>Provider</th><th>显示名称</th><th>Context Window</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="r in routes" :key="r.alias">
            <td><span class="mono" style="color:var(--color-primary); font-weight:600;">{{ r.alias }}</span></td>
            <td class="mono">{{ r.model }}</td>
            <td>
              <span class="tag tag-success">{{ r.provider }}</span>
            </td>
            <td>{{ r.display_name || '—' }}</td>
            <td class="mono">{{ r.context_window ? fmtCtx(r.context_window) : '—' }}</td>
            <td>
              <button class="btn btn-ghost btn-sm" @click="openEdit(r)">编辑</button>
              <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" @click="confirmDelete(r)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Drawer -->
    <div v-if="showDrawer" class="drawer-overlay" @click.self="showDrawer=false">
      <div class="drawer-content">
        <h2 class="modal-title">{{ editingAlias ? '编辑路由' : '添加路由' }}</h2>

        <div class="form-group">
          <label class="form-label">Alias *</label>
          <input class="form-input" v-model="form.alias" :disabled="!!editingAlias" placeholder="moonbridge" />
          <div class="text-secondary" style="font-size:11px; margin-top:4px;">客户端使用的模型别名</div>
        </div>

        <div class="form-group">
          <label class="form-label">目标模型 *</label>
          <div class="flex items-center gap-2">
            <select class="form-select" v-model="form.model">
              <option value="" disabled>选择可用模型...</option>
              <option v-for="m in availableModels" :key="m.slug" :value="m.slug">
                {{ m.slug }} {{ m.display_name ? '(' + m.display_name + ')' : '' }}
                <template v-if="m.providers?.length"> — {{ m.providers.join(', ') }}</template>
              </option>
            </select>
            <router-link to="/models" class="btn btn-ghost btn-sm" style="flex-shrink:0;">新建</router-link>
          </div>
          <div v-if="form.model && !availableModels.find(m => m.slug === form.model)" class="form-error">
            此模型未关联任何 Provider，无法创建路由
          </div>
        </div>

        <!-- Provider: 单选自动填充，多选展示下拉 -->
        <div class="form-group" v-if="form.model && availableProvidersForModel.length > 1">
          <label class="form-label">Provider *</label>
          <div class="flex items-center gap-2">
            <select class="form-select" v-model="form.provider">
              <option value="" disabled>选择 Provider...</option>
              <option v-for="p in availableProvidersForModel" :key="p.key" :value="p.key">{{ p.key }}</option>
            </select>
          </div>
        </div>
        <div v-else-if="form.model && availableProvidersForModel.length === 1" class="form-group">
          <label class="form-label">Provider</label>
          <div class="flex items-center gap-2" style="padding: 9px 12px; background: var(--bg-input); border: 1px solid var(--border-color); border-radius: var(--border-radius-sm); color: var(--text-secondary); font-size: 13px;">
            <span>{{ form.provider || '加载中...' }}</span>
            <span class="tag tag-success">自动匹配</span>
          </div>
        </div>

        <div class="form-group">
          <label class="form-label">Display Name</label>
          <input class="form-input" v-model="form.display_name" placeholder="Moon Bridge Pro" />
        </div>

        <div class="form-group">
          <label class="form-label">Context Window</label>
          <input class="form-input" type="number" v-model.number="form.context_window" placeholder="留空则继承模型定义" />
        </div>

        <p class="text-secondary" style="font-size:11px; margin-top:16px;">
          💡 路由高级配置（模型定价覆盖、推理等级、输入模态、Web Search、扩展等）请至 <router-link to="/config">配置管理 → YAML 编辑器</router-link> 编辑
        </p>

        <div class="modal-actions" style="border-top:none; padding-top:0;">
          <button class="btn btn-ghost w-full" @click="showDrawer=false">取消</button>
          <button class="btn btn-primary w-full" @click="saveRoute" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <!-- Delete Confirm -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget=null">
      <div class="modal-content" style="max-width:400px;">
        <h2 class="modal-title">确认删除</h2>
        <p>确定要删除路由 <strong>{{ deleteTarget.alias }}</strong> 吗？</p>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="deleteTarget=null">取消</button>
          <button class="btn btn-danger" @click="doDelete">删除</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, inject } from 'vue'
import { listRoutes, createRoute, deleteRoute, listModels, listProviders } from '../api/index.js'
import { usePostSave } from '../composables/useToast.js'

const showToast = inject('showToast')
const showPostSave = usePostSave()

const loading = ref(true)
const error = ref(null)
const routes = ref([])
const models = ref([])
const providers = ref([])

const showDrawer = ref(false)
const editingAlias = ref('')
const saving = ref(false)
const form = ref({ alias: '', model: '', provider: '', display_name: '', context_window: null })
const deleteTarget = ref(null)

// 只显示已关联至少一个 Provider 的可用模型
const availableModels = computed(() => {
  return models.value.filter(m => m.providers && m.providers.length > 0)
})

// 当前选中模型的可选 Provider 列表
const availableProvidersForModel = computed(() => {
  if (!form.value.model) return []
  const m = models.value.find(m => m.slug === form.value.model)
  if (!m || !m.providers) return []
  return providers.value.filter(p => m.providers.includes(p.key))
})

// 模型变化时自动设置 Provider（单一时自动填充，多个时清空让用户选择）
watch(() => form.value.model, (newModel) => {
  if (!newModel) { form.value.provider = ''; return }
  const m = models.value.find(m => m.slug === newModel)
  if (!m || !m.providers) { form.value.provider = ''; return }
  if (m.providers.length === 1) {
    form.value.provider = m.providers[0]
  } else if (editingAlias.value) {
    // 编辑模式：保持原有 provider 不变
  } else {
    form.value.provider = ''
  }
})

function fmtCtx(n) { return n ? (n >= 1000 ? (n / 1000).toFixed(0) + 'K' : String(n)) : '—' }

function openCreate() {
  editingAlias.value = ''
  form.value = { alias: '', model: '', provider: '', display_name: '', context_window: null }
  showDrawer.value = true
}

function openEdit(r) {
  editingAlias.value = r.alias
  form.value = {
    alias: r.alias,
    model: r.model,
    provider: r.provider,
    display_name: r.display_name || '',
    context_window: r.context_window || null,
  }
  showDrawer.value = true
}

async function saveRoute() {
  if (!form.value.alias || !form.value.model || !form.value.provider) {
    showToast('Alias、模型和 Provider 为必填项', 'warning')
    return
  }
  saving.value = true
  try {
    await createRoute(form.value.alias, {
      model: form.value.model,
      provider: form.value.provider,
      display_name: form.value.display_name || undefined,
      context_window: form.value.context_window || undefined,
    })
    showPostSave()
    showToast(`路由 "${form.value.alias}" 已暂存${editingAlias.value ? '更新' : '创建'}`, 'success')
    showDrawer.value = false
    fetchData()
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally { saving.value = false }
}

function confirmDelete(r) { deleteTarget.value = r }

async function doDelete() {
  if (!deleteTarget.value) return
  try {
    await deleteRoute(deleteTarget.value.alias)
    showPostSave()
    showToast(`路由 "${deleteTarget.value.alias}" 删除已暂存`, 'success')
    deleteTarget.value = null
    fetchData()
  } catch (e) {
    showToast(`删除失败: ${e.message}`, 'error')
  }
}

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    const [rRes, mRes, pRes] = await Promise.all([
      listRoutes({ limit: 100 }),
      listModels({ limit: 100 }),
      listProviders({ limit: 100 }),
    ])
    routes.value = rRes.data || []
    models.value = mRes.data || []
    providers.value = pRes.data || []
  } catch (e) {
    error.value = e.message
  } finally { loading.value = false }
}

onMounted(fetchData)
</script>
