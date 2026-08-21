import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api } from '../services/api'

export interface Claim {
  id: string
  claimant_id: string
  project_id: string
  category: string
  occurred_on: string
  amount: string
  status: string
  version: number
}

interface ClaimPage {
  items: Claim[]
  page: number
  page_size: number
  total: number
}

export const useWorkspaceStore = defineStore('workspace', () => {
  const claims = ref<Claim[]>([])
  const selectedClaimId = ref('')
  const loading = ref(false)
  const error = ref('')
  const activeView = ref('workbench')
  const total = ref(0)

  const selectedClaim = computed(() => claims.value.find(item => item.id === selectedClaimId.value))

  async function loadClaims(status = '') {
    loading.value = true
    error.value = ''
    try {
      const query = new URLSearchParams({ page: '1', page_size: '50', sort_by: 'created_at', sort_order: 'desc' })
      if (status) query.set('status', status)
      const page = await api<ClaimPage>(`/claims?${query}`)
      claims.value = page.items
      total.value = page.total
      if (!selectedClaimId.value && page.items.length) selectedClaimId.value = page.items[0].id
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : '加载失败'
    } finally {
      loading.value = false
    }
  }

  function selectView(view: string) {
    activeView.value = view
  }

  return { claims, selectedClaimId, selectedClaim, activeView, total, loading, error, loadClaims, selectView }
})

