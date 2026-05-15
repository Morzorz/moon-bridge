<template>
  <div>
    <div class="flex items-center justify-between mb-4">
      <input class="form-input" style="width: 260px;" v-model="search" placeholder="搜索模型 slug..." @input="doSearch" />
      <button class="btn btn-primary" @click="openCreate">＋ 添加模型</button>
    </div>

    <div v-if="loading" class="card"><div class="skeleton" style="height:200px;"></div></div>

    <div v-else-if="error" class="card" style="border-color:var(--color-danger);">
      <p style="color:var(--color-danger);">{{ error }}</p>
      <button class="btn btn-ghost btn-sm mt-2" @click="fetchData">重试</button>
    </div>

    <div v-else-if="!filteredModels.length" class="empty-state">
      <div class="empty-icon">🧩</div>
      <h3>暂无模型定义</h3>
      <p>添加模型后可在 Provider 中关联定价 Offer</p>
      <button class="btn btn-primary" @click="openCreate">添加模型</button>
    </div>

    <div v-else class="card" style="padding:0; overflow:hidden;">
      <table class="data-table">
        <thead>
          <tr>
            <th @click="sortBy('slug')" :class="{sorted: sortKey==='slug'}">Slug {{ sortIcon('slug') }}</th>
            <th @click="sortBy('display_name')" :class="{sorted: sortKey==='display_name'}">显示名称</th>
            <th @click="sortBy('context_window')" :class="{sorted: sortKey==='context_window'}">Context Window</th>
            <th>关联 Provider</th>
            <th style="width:100px;">操作</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="m in sortedModels" :key="m.slug">
            <tr class="clickable" @click="toggleExpand(m.slug)">
              <td class="mono"><strong>{{ m.slug }}</strong></td>
              <td>{{ m.display_name || '—' }}</td>
              <td class="mono">{{ fmtCtx(m.context_window) }}</td>
              <td>
                <span v-for="p in m.providers" :key="p" class="tag tag-primary" style="margin-right:4px; margin-bottom:2px;">{{ p }}</span>
                <span v-if="!m.providers.length" class="text-secondary">无</span>
              </td>
              <td>
                <button class="btn btn-ghost btn-sm" @click.stop="openEdit(m)">编辑</button>
                <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" @click.stop="confirmDelete(m)">删除</button>
              </td>
            </tr>
            <!-- Expanded: 关联 Provider + Offers 管理 -->
            <tr v-if="expanded === m.slug">
              <td colspan="5" style="padding:0; background:var(--bg-tertiary);">
                <div style="padding:16px 20px;">

                  <!-- === 已关联 Providers === -->
                  <div class="flex items-center justify-between mb-3">
                    <span class="section-title" style="margin:0;">🔗 已关联 Providers</span>
                  </div>

                  <table class="data-table" v-if="(offerMap[m.slug] || []).length">
                    <thead>
                      <tr>
                        <th>Provider</th><th>上游模型名</th><th>优先级</th><th>Input Price</th><th>Output Price</th><th>Cache Write</th><th>Cache Read</th><th>操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="o in (offerMap[m.slug] || [])" :key="o.provider_key + '-' + o.model">
                        <td class="mono"><strong>{{ o.provider_key }}</strong></td>
                        <td class="mono">{{ o.upstream_name || o.model }}</td>
                        <td>{{ o.priority ?? '—' }}</td>
                        <td class="mono">¥{{ (o.input_price || 0).toFixed(2) }}</td>
                        <td class="mono">¥{{ (o.output_price || 0).toFixed(2) }}</td>
                        <td class="mono">¥{{ (o.cache_write || 0).toFixed(2) }}</td>
                        <td class="mono">¥{{ (o.cache_read || 0).toFixed(2) }}</td>
                        <td>
                          <button class="btn btn-ghost btn-sm" @click="editOffer(o)">编辑</button>
                          <button class="btn btn-ghost btn-sm" style="color:var(--color-danger);" @click="deleteOfferItem(o)">删除</button>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                  <div v-else class="text-secondary" style="padding:8px 0 16px;">暂未关联任何 Provider</div>

                  <!-- === 未关联 Providers === -->
                  <div v-if="unassociated(m.slug).length" class="mt-4">
                    <span class="section-title" style="margin-bottom:12px;">＋ 未关联 Providers</span>
                    <div class="flex" style="gap:8px; flex-wrap: wrap;">
                      <div v-for="pk in unassociated(m.slug)" :key="pk"
                        class="card" style="padding:10px 14px; display:flex; align-items:center; gap:10px;">
                        <span class="mono"><strong>{{ pk }}</strong></span>
                        <button class="btn btn-primary btn-sm" @click="openQuickLink(m.slug, pk)">＋ 关联</button>
                      </div>
                    </div>
                  </div>
                  <div v-else class="text-secondary mt-4" style="font-size:12px;">
                    所有 Provider 均已关联此模型
                  </div>

                  <!-- Quick link inline form -->
                  <div v-if="quickLinkSlug === m.slug && quickLinkProvider" class="card mt-3" style="background:var(--bg-primary);">
                    <div class="flex items-center gap-2 mb-3">
                      <span class="tag tag-success">快速关联</span>
                      <strong>{{ quickLinkProvider }} → {{ m.slug }}</strong>
                    </div>
                    <div class="form-row">
                      <div class="form-group">
                        <label class="form-label">Input Price (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="quickForm.input_price" />
                      </div>
                      <div class="form-group">
                        <label class="form-label">Output Price (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="quickForm.output_price" />
                      </div>
                    </div>
                    <div class="form-row">
                      <div class="form-group">
                        <label class="form-label">Cache Write (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="quickForm.cache_write" />
                      </div>
                      <div class="form-group">
                        <label class="form-label">Cache Read (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="quickForm.cache_read" />
                      </div>
                    </div>
                    <div class="form-group">
                      <label class="form-label">上游模型名（可选）</label>
                      <input class="form-input" v-model="quickForm.upstream_name" placeholder="留空则与 slug 相同" />
                    </div>
                    <div class="flex gap-2">
                      <button class="btn btn-primary btn-sm" @click="saveQuickLink" :disabled="savingQuick">
                        {{ savingQuick ? '保存中...' : '确认关联' }}
                      </button>
                      <button class="btn btn-ghost btn-sm" @click="cancelQuickLink">取消</button>
                    </div>
                  </div>

                  <!-- Inline edit form -->
                  <div v-if="editTargetKey === m.slug && editTargetProvider" class="card mt-3" style="background:var(--bg-primary);">
                    <div class="flex items-center gap-2 mb-3">
                      <span class="tag tag-primary">编辑 Offer</span>
                      <strong>{{ editTargetProvider }} → {{ m.slug }}</strong>
                    </div>
                    <div class="form-row">
                      <div class="form-group">
                        <label class="form-label">Input Price (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="editForm.input_price" />
                      </div>
                      <div class="form-group">
                        <label class="form-label">Output Price (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="editForm.output_price" />
                      </div>
                    </div>
                    <div class="form-row">
                      <div class="form-group">
                        <label class="form-label">Cache Write (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="editForm.cache_write" />
                      </div>
                      <div class="form-group">
                        <label class="form-label">Cache Read (¥/M)</label>
                        <input class="form-input" type="number" step="0.01" v-model.number="editForm.cache_read" />
                      </div>
                    </div>
                    <div class="form-row">
                      <div class="form-group">
                        <label class="form-label">优先级</label>
                        <input class="form-input" type="number" v-model.number="editForm.priority" />
                      </div>
                      <div class="form-group">
                        <label class="form-label">上游模型名</label>
                        <input class="form-input" v-model="editForm.upstream_name" placeholder="留空则与 slug 相同" />
                      </div>
                    </div>
                    <div class="flex gap-2">
                      <button class="btn btn-primary btn-sm" @click="saveEditOffer" :disabled="savingEdit">
                        {{ savingEdit ? '保存中...' : '保存修改' }}
                      </button>
                      <button class="btn btn-ghost btn-sm" @click="cancelEditOffer">取消</button>
                    </div>
                  </div>

                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <!-- Create/Edit Model Modal -->
    <div v-if="showModal" class="modal-overlay" @click.self="closeModal">
      <div class="modal-content">
        <h2 class="modal-title">{{ editingSlug ? '编辑模型' : '添加模型' }}</h2>
        <div class="form-group">
          <label class="form-label">Slug *</label>
          <input class="form-input" v-model="form.slug" :disabled="!!editingSlug" placeholder="deepseek-v4-pro" />
        </div>
        <div class="form-group">
          <label class="form-label">Display Name</label>
          <input class="form-input" v-model="form.display_name" placeholder="DeepSeek V4 Pro" />
        </div>
        <div class="form-group">
          <label class="form-label">Context Window</label>
          <input class="form-input" type="number" v-model.number="form.context_window" placeholder="128000" />
        </div>
        <div class="form-group">
          <label class="form-label">Max Output Tokens</label>
          <input class="form-input" type="number" v-model.number="form.max_output_tokens" placeholder="384000" />
        </div>
        <div class="form-group">
          <label class="form-label">Description</label>
          <textarea class="form-textarea" rows="2" v-model="form.description" placeholder="简短描述此模型"></textarea>
        </div>
        <p class="text-secondary" style="font-size:11px; margin-bottom:12px;">
          💡 高级配置（推理等级、输入模态、扩展等）请至 <router-link to="/config">配置管理 → YAML 编辑器</router-link> 编辑
        </p>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeModal">取消</button>
          <button class="btn btn-primary" @click="saveModel" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <!-- Add Offer Modal -->
    <div v-if="showOfferModal" class="modal-overlay" @click.self="showOfferModal=false">
      <div class="modal-content" style="min-width:540px;">
        <h2 class="modal-title">添加 Offer</h2>
        <div class="form-group">
          <label class="form-label">Provider *</label>
          <select class="form-select" v-model="offerForm.provider_key">
            <option v-for="p in allProviders" :key="p.key" :value="p.key">{{ p.key }}</option>
          </select>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Input Price (¥/M)</label>
            <input class="form-input" type="number" step="0.01" v-model.number="offerForm.input_price" />
          </div>
          <div class="form-group">
            <label class="form-label">Output Price (¥/M)</label>
            <input class="form-input" type="number" step="0.01" v-model.number="offerForm.output_price" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">Cache Write (¥/M)</label>
            <input class="form-input" type="number" step="0.01" v-model.number="offerForm.cache_write" />
          </div>
          <div class="form-group">
            <label class="form-label">Cache Read (¥/M)</label>
            <input class="form-input" type="number" step="0.01" v-model.number="offerForm.cache_read" />
          </div>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">优先级 (Priority)</label>
            <input class="form-input" type="number" v-model.number="offerForm.priority" placeholder="0 = 最高优先级" />
          </div>
          <div class="form-group">
            <label class="form-label">上游模型名</label>
            <input class="form-input" v-model="offerForm.upstream_name" placeholder="留空则与 slug 相同" />
          </div>
        </div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="showOfferModal=false">取消</button>
          <button class="btn btn-primary" @click="saveOffer" :disabled="savingOffer">{{ savingOffer ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <!-- Delete Confirm -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget=null">
      <div class="modal-content" style="max-width:400px;">
        <h2 class="modal-title">确认删除</h2>
        <p>确定要删除模型 <strong>{{ deleteTarget.slug }}</strong> 吗？</p>
        <p class="text-secondary mt-2" style="font-size:12px;">如仍有 Provider 引用此模型则无法删除。</p>
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
  listModels, createModel, deleteModel,
  listProviders, getProvider, createOffer, updateOffer, deleteOffer,
} from '../api/index.js'
const showToast = inject('showToast')

const loading = ref(true)
const error = ref(null)
const models = ref([])
const allProviders = ref([])
const search = ref('')
const sortKey = ref('slug')
const sortDir = ref(1)
const expanded = ref(null)

// Offer data: map slug -> offers[]
const offerMap = ref({})

// Modal states
const showModal = ref(false)
const editingSlug = ref('')
const saving = ref(false)
const form = ref({ slug: '', display_name: '', context_window: 128000 })

const showOfferModal = ref(false)
const offerFormSlug = ref('')
const savingOffer = ref(false)
const offerForm = ref({ provider_key: '', input_price: 0, output_price: 0, cache_write: 0, cache_read: 0, upstream_name: '', priority: 0 })

// Quick link (inline association)
const quickLinkSlug = ref(null)
const quickLinkProvider = ref(null)
const savingQuick = ref(false)
const quickForm = ref({ input_price: 0, output_price: 0, cache_write: 0, cache_read: 0, upstream_name: '' })

// Inline edit
const editTargetKey = ref(null)
const editTargetProvider = ref(null)
const savingEdit = ref(false)
const editForm = ref({ input_price: 0, output_price: 0, cache_write: 0, cache_read: 0, priority: 0, upstream_name: '' })

const deleteTarget = ref(null)

const filteredModels = computed(() => {
  let list = models.value
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(m => m.slug.toLowerCase().includes(q) || (m.display_name || '').toLowerCase().includes(q))
  }
  return list
})

const sortedModels = computed(() => {
  const list = [...filteredModels.value]
  list.sort((a, b) => {
    let va = a[sortKey.value], vb = b[sortKey.value]
    if (typeof va === 'string') va = va.toLowerCase()
    if (typeof vb === 'string') vb = vb.toLowerCase()
    if (va < vb) return -1 * sortDir.value
    if (va > vb) return 1 * sortDir.value
    return 0
  })
  return list
})

function sortBy(key) {
  if (sortKey.value === key) sortDir.value *= -1
  else { sortKey.value = key; sortDir.value = 1 }
}

function sortIcon(key) {
  if (sortKey.value !== key) return ''
  return sortDir.value === 1 ? '↑' : '↓'
}

function doSearch() {}

function fmtCtx(n) { return n ? (n >= 1000 ? (n / 1000).toFixed(0) + 'K' : String(n)) : '—' }

/** Load offer details for a single model by fetching all provider details */
async function fetchOffers(slug) {
  try {
    const provs = allProviders.value
    const acc = []
    for (const p of provs) {
      try {
        const detail = await getProvider(p.key)
        const providerKey = detail.key || p.key
        for (const o of (detail.offers || [])) {
          if (o.model === slug) {
            acc.push({ provider_key: providerKey, ...o })
          }
        }
      } catch { /* skip unreachable providers */ }
    }
    offerMap.value = { ...offerMap.value, [slug]: acc }
  } catch {}
}

/** Preload all provider details and build the full offer map */
async function preloadOffers() {
  const provs = allProviders.value
  const map = {}
  for (const p of provs) {
    try {
      const detail = await getProvider(p.key)
      const providerKey = detail.key || p.key
      for (const o of (detail.offers || [])) {
        const slug = o.model
        if (!map[slug]) map[slug] = []
        map[slug].push({ provider_key: providerKey, ...o })
      }
    } catch { /* skip unreachable providers */ }
  }
  offerMap.value = map
}

function toggleExpand(slug) {
  if (expanded.value === slug) {
    expanded.value = null
    return
  }
  expanded.value = slug
  // Load offers on first expand for this slug
  if (!offerMap.value[slug]) {
    fetchOffers(slug)
  }
}

function openCreate() {
  editingSlug.value = ''
  form.value = { slug: '', display_name: '', context_window: 128000, max_output_tokens: null, description: '' }
  showModal.value = true
}

async function openEdit(m) {
  editingSlug.value = m.slug
  try {
    const detail = await getModel(m.slug)
    form.value = {
      slug: detail.slug,
      display_name: detail.display_name || '',
      context_window: detail.context_window || 128000,
      max_output_tokens: detail.max_output_tokens || null,
      description: detail.description || '',
    }
  } catch {
    form.value = { slug: m.slug, display_name: m.display_name || '', context_window: m.context_window || 128000, max_output_tokens: null, description: '' }
  }
  showModal.value = true
}

function closeModal() { showModal.value = false }

async function saveModel() {
  if (!form.value.slug) { showToast('Slug 为必填项', 'warning'); return }
  saving.value = true
  try {
    await createModel(form.value.slug, {
      display_name: form.value.display_name || undefined,
      context_window: form.value.context_window || undefined,
      max_output_tokens: form.value.max_output_tokens || undefined,
      description: form.value.description || undefined,
    })
    showToast(`模型 "${form.value.slug}" 已保存并生效`, 'success')
    closeModal()
    fetchData()
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally { saving.value = false }
}

function confirmDelete(m) { deleteTarget.value = m }

async function doDelete() {
  if (!deleteTarget.value) return
  try {
    await deleteModel(deleteTarget.value.slug)
    showToast(`模型 "${deleteTarget.value.slug}" 已删除`, 'success')
    deleteTarget.value = null
    fetchData()
  } catch (e) {
    showToast(`删除失败: ${e.message}`, 'error')
  }
}

function openAddOffer(slug) {
  offerFormSlug.value = slug
  offerForm.value = { provider_key: allProviders.value[0]?.key || '', input_price: 0, output_price: 0, cache_write: 0, cache_read: 0, priority: 0, upstream_name: '' }
  showOfferModal.value = true
}

async function saveOffer() {
  if (!offerForm.value.provider_key) { showToast('请选择 Provider', 'warning'); return }
  savingOffer.value = true
  try {
    await createOffer(offerForm.value.provider_key, {
      model: offerFormSlug.value,
      upstream_name: offerForm.value.upstream_name || undefined,
      input_price: offerForm.value.input_price || undefined,
      output_price: offerForm.value.output_price || undefined,
      cache_write: offerForm.value.cache_write || undefined,
      cache_read: offerForm.value.cache_read || undefined,
      priority: offerForm.value.priority ?? undefined,
    })
    showToast('Offer 已保存并生效', 'success')
    showOfferModal.value = false
  } catch (e) {
    showToast(`保存失败: ${e.message}`, 'error')
  } finally { savingOffer.value = false }
}

/** Compute providers NOT yet offering this model slug */
function unassociated(slug) {
  const offered = new Set((offerMap.value[slug] || []).map(o => o.provider_key))
  return allProviders.value.filter(p => !offered.has(p.key)).map(p => p.key)
}

// === Quick link (create offer inline) ===
function openQuickLink(slug, providerKey) {
  cancelEditOffer()
  quickLinkSlug.value = slug
  quickLinkProvider.value = providerKey
  quickForm.value = { input_price: 0, output_price: 0, cache_write: 0, cache_read: 0, upstream_name: '' }
}
function cancelQuickLink() {
  quickLinkSlug.value = null
  quickLinkProvider.value = null
}
async function saveQuickLink() {
  if (!quickLinkProvider.value) return
  savingQuick.value = true
  try {
    await createOffer(quickLinkProvider.value, {
      model: quickLinkSlug.value,
      upstream_name: quickForm.value.upstream_name || undefined,
      input_price: quickForm.value.input_price || undefined,
      output_price: quickForm.value.output_price || undefined,
      cache_write: quickForm.value.cache_write || undefined,
      cache_read: quickForm.value.cache_read || undefined,
    })
    showToast(`✅ "${quickLinkProvider.value}" 已关联 "${quickLinkSlug.value}"`, 'success')
    cancelQuickLink()
    fetchOffers(quickLinkSlug.value)
  } catch (e) { showToast(`关联失败: ${e.message}`, 'error')
  } finally { savingQuick.value = false }
}

// === Inline edit offer ===
function editOffer(o) {
  cancelQuickLink()
  editTargetKey.value = o.model
  editTargetProvider.value = o.provider_key
  editForm.value = {
    input_price: o.input_price ?? 0,
    output_price: o.output_price ?? 0,
    cache_write: o.cache_write ?? 0,
    cache_read: o.cache_read ?? 0,
    priority: o.priority ?? 0,
    upstream_name: o.upstream_name || '',
  }
}
function cancelEditOffer() {
  editTargetKey.value = null
  editTargetProvider.value = null
}
async function saveEditOffer() {
  if (!editTargetProvider.value) return
  savingEdit.value = true
  try {
    await updateOffer(editTargetProvider.value, editTargetKey.value, {
      input_price: editForm.value.input_price ?? undefined,
      output_price: editForm.value.output_price ?? undefined,
      cache_write: editForm.value.cache_write ?? undefined,
      cache_read: editForm.value.cache_read ?? undefined,
      priority: editForm.value.priority ?? undefined,
      upstream_name: editForm.value.upstream_name || undefined,
    })
    showToast(`✅ "${editTargetProvider.value}" 的 Offer 已更新`, 'success')
    const slug = editTargetKey.value
    cancelEditOffer()
    fetchOffers(slug)
  } catch (e) { showToast(`更新失败: ${e.message}`, 'error')
  } finally { savingEdit.value = false }
}

async function deleteOfferItem(o) {
  try {
    await deleteOffer(o.provider_key, o.model)
    showToast('Offer 已删除', 'success')
  } catch (e) { showToast(`删除失败: ${e.message}`, 'error') }
}

async function fetchData() {
  loading.value = true
  error.value = null
  try {
    const [mRes, pRes] = await Promise.all([listModels({ limit: 100 }), listProviders({ limit: 100 })])
    models.value = mRes.data || []
    allProviders.value = pRes.data || []
    // Preload offer data in background
    if (allProviders.value.length > 0) {
      preloadOffers()
    }
  } catch (e) {
    error.value = e.message
  } finally { loading.value = false }
}

onMounted(fetchData)
</script>
