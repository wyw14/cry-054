<script setup lang="ts">
import type { Claim } from '../stores/workspace'

defineProps<{ claims: Claim[]; selected: string; loading: boolean }>()
defineEmits<{ select: [id: string] }>()

const statusName: Record<string, string> = {
  submitted: '待分派', under_review: '复核中', need_supplement: '待补件',
  approved: '已通过', settled: '已结算', rejected: '已退回',
}
</script>

<template>
  <section class="queue">
    <header><div><h2>复核队列</h2><p>按发生日期和风险标记排序</p></div><button class="icon-button" title="刷新">↻</button></header>
    <div v-if="loading" class="empty">正在加载…</div>
    <div v-else-if="!claims.length" class="empty">当前没有待处理申报</div>
    <button
      v-for="claim in claims" :key="claim.id"
      class="queue-row" :class="{ selected: claim.id === selected }"
      @click="$emit('select', claim.id)"
    >
      <span class="claim-title">{{ claim.category }} · {{ claim.id }}</span>
      <span class="money">¥ {{ claim.amount }}</span>
      <small>{{ claim.occurred_on.slice(0, 10) }}</small>
      <span class="status">{{ statusName[claim.status] ?? claim.status }}</span>
    </button>
  </section>
</template>

