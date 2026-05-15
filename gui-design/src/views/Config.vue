<template>
  <div class="card-grid card-grid-2" style="grid-template-columns: 280px 1fr;">
    <!-- Left: Operations -->
    <div>
      <div class="card" style="margin-bottom: 16px;">
        <div class="section-title">当前配置信息</div>
        <div class="text-secondary" style="font-size: 13px; line-height: 2;">
          <div>版本: <span class="text-mono">{{ configInfo.version || '—' }}</span></div>
          <div>Providers: <strong>{{ configInfo.provider_count }}</strong></div>
          <div>Routes: <strong>{{ configInfo.route_count }}</strong></div>
          <div>模式: <span class="tag tag-primary">{{ configInfo.mode }}</span></div>
        </div>
      </div>

      <div class="card" style="margin-bottom: 16px;">
        <button class="btn btn-primary w-full mb-2" @click="doExport" :disabled="exporting">
          {{ exporting ? '导出中...' : '↓ 导出 YAML' }}
        </button>
        <button class="btn btn-ghost w-full mb-2" @click="validateMode = true">
          ✓ 验证配置
        </button>
        <button class="btn btn-success w-full" @click="doImport" :disabled="importing">
          {{ importing ? '导入中...' : '↑ 导入配置' }}
        </button>
      </div>

      <!-- Validation result -->
      <div v-if="validateResult" class="card" :style="{ borderColor: validateResult.valid ? 'var(--color-success)' : 'var(--color-danger)' }">
        <div v-if="validateResult.valid" style="color:var(--color-success);">✅ 配置格式正确</div>
        <div v-else>
          <div style="color:var(--color-danger); margin-bottom:8px;">❌ 验证失败:</div>
          <div v-for="(err, i) in (validateResult.errors || [])" :key="i" class="text-mono" style="font-size:12px; color:var(--color-danger); padding:2px 0;">
            {{ typeof err === 'string' ? err : (err.path + ': ' + err.message) }}
          </div>
        </div>
      </div>
    </div>

    <!-- Right: YAML Editor -->
    <div class="card" style="padding:0; overflow:hidden; display:flex; flex-direction:column;">
      <div class="flex items-center justify-between" style="padding:12px 16px; border-bottom: 1px solid var(--border-color);">
        <div class="flex gap-2">
          <button
            class="btn btn-ghost btn-sm"
            :class="{ active: mode === 'view' }"
            @click="mode = 'view'; loadEffective()"
            :style="mode==='view' ? { background:'var(--color-primary-bg)', color:'var(--color-primary)' } : {}"
          >查看</button>
          <button
            class="btn btn-ghost btn-sm"
            :class="{ active: mode === 'edit' }"
            @click="mode = 'edit'"
            :style="mode==='edit' ? { background:'var(--color-primary-bg)', color:'var(--color-primary)' } : {}"
          >编辑</button>
        </div>
        <div class="text-secondary" style="font-size:12px;">YAML</div>
      </div>

      <textarea
        class="form-textarea"
        style="flex:1; border:none; border-radius:0; min-height:500px; font-size:12.5px; line-height:1.7; resize:none;"
        v-model="yamlContent"
        :readonly="mode === 'view'"
        placeholder="在此粘贴 YAML 配置..."
        spellcheck="false"
      ></textarea>
    </div>
  </div>

  <!-- Import confirm -->
  <div v-if="showImportConfirm" class="modal-overlay" @click.self="showImportConfirm=false">
    <div class="modal-content" style="max-width:480px;">
      <h2 class="modal-title">确认导入</h2>
      <p>确定要用新的配置覆盖当前配置吗？</p>
      <p class="text-secondary mt-2" style="font-size:12px;">导入后会暂存为 pending change，需在变更管理页应用。</p>
      <div class="modal-actions">
        <button class="btn btn-ghost" @click="showImportConfirm=false">取消</button>
        <button class="btn btn-success" @click="confirmImport" :disabled="importing">
          {{ importing ? '导入中...' : '确认导入' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, inject } from 'vue'
import { getStatus, exportConfigRaw, importConfig, validateConfig } from '../api/index.js'


const showToast = inject('showToast')

const configInfo = ref({})
const yamlContent = ref('')
const mode = ref('view')
const validateResult = ref(null)
const validateMode = ref(false)
const showImportConfirm = ref(false)
const exporting = ref(false)
const importing = ref(false)

async function loadEffective() {
  try {
    const yaml = await exportConfigRaw()
    yamlContent.value = yaml
    mode.value = 'view'
  } catch (e) {
    showToast(`加载配置失败: ${e.message}`, 'error')
  }
}

async function doExport() {
  exporting.value = true
  try {
    const yaml = await exportConfigRaw()
    // If response is a string (YAML), use it directly; if object, stringify
    const content = typeof yaml === 'string' ? yaml : JSON.stringify(yaml, null, 2)
    yamlContent.value = content
    mode.value = 'view'
    showToast('配置已加载到编辑器', 'success')
  } catch (e) {
    showToast(`导出失败: ${e.message}`, 'error')
  } finally { exporting.value = false }
}

function doImport() {
  if (!yamlContent.value.trim()) {
    showToast('请先在编辑器中粘贴 YAML 配置', 'warning')
    return
  }
  showImportConfirm.value = true
}

async function confirmImport() {
  importing.value = true
  try {
    const res = await importConfig(yamlContent.value)
    showToast('配置已导入并生效', 'success')
    showImportConfirm.value = false
  } catch (e) {
    showToast(`导入失败: ${e.message}`, 'error')
  } finally { importing.value = false }
}

async function doValidate() {
  if (!yamlContent.value.trim()) {
    showToast('请先在编辑器中粘贴 YAML 配置', 'warning')
    return
  }
  try {
    const res = await validateConfig(yamlContent.value)
    validateResult.value = res
    if (res.valid) showToast('配置验证通过 ✅', 'success')
  } catch (e) {
    showToast(`验证请求失败: ${e.message}`, 'error')
  }
}

// When validateMode becomes true, trigger validation
import { watch } from 'vue'
watch(validateMode, (val) => {
  if (val) { doValidate(); validateMode.value = false }
})

onMounted(async () => {
  try {
    const st = await getStatus()
    configInfo.value = st
  } catch {}
  await loadEffective()
})
</script>
