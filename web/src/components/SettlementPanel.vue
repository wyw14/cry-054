<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Claim } from '../stores/workspace'
import { api } from '../services/api'

const props = defineProps<{ claim?: Claim }>()
const preview = ref<Record<string, unknown> | null>(null)
const busy = ref(false)
const message = ref('')

const claimTitle = computed(() => props.claim ? `${props.claim.category} · ${props.claim.id}` : '请选择申报')

async function runPreview() {
  if (!props.claim) return
  busy.value = true
  message.value = ''
  try {
    preview.value = await api<Record<string, unknown>>(`/claims/${props.claim.id}/preview`, { method: 'POST' })
  } catch (cause) {
    message.value = cause instanceof Error ? cause.message : '试算失败'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="settlement">
    <header><div><p class="eyebrow">结算试算</p><h2>{{ claimTitle }}</h2></div><button :disabled="!claim || busy" @click="runPreview">{{ busy ? '计算中' : '重新试算' }}</button></header>
    <div v-if="!claim" class="empty">从左侧选择一条申报查看规则解释</div>
    <template v-else>
      <div class="summary-grid">
        <div><span>申报金额</span><strong>¥ {{ claim.amount }}</strong></div>
        <div><span>当前状态</span><strong>{{ claim.status }}</strong></div>
        <div><span>版本</span><strong>v{{ claim.version }}</strong></div>
      </div>
      <p v-if="message" class="inline-error">{{ message }}</p>
      <pre v-if="preview" class="preview-json">{{ JSON.stringify(preview, null, 2) }}</pre>
      <div v-else class="explanation-placeholder">
        <span>规则版本、分段比例和年度占用会在试算后逐项解释。</span>
      </div>
    </template>
  </section>
</template>

