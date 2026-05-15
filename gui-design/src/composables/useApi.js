import { ref } from 'vue'

/**
 * 通用 API 调用组合式函数，封装 loading / error / data 三板斧
 */
export function useApi(fetchFn) {
  const data = ref(null)
  const loading = ref(false)
  const error = ref(null)

  async function execute(...args) {
    loading.value = true
    error.value = null
    try {
      data.value = await fetchFn(...args)
    } catch (e) {
      error.value = e.message || String(e)
    } finally {
      loading.value = false
    }
    return { data: data.value, error: error.value }
  }

  return { data, loading, error, execute }
}
