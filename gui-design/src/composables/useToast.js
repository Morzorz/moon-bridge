import { inject } from 'vue'

export function useToast() {
  return inject('showToast')
}

export function usePostSave() {
  return inject('showPostSave')
}
