<script setup lang="ts">
defineProps<{ pending: number; exceptions: number; occupied: string; remaining: string }>()
</script>

<template>
  <section class="metrics" aria-label="年度运行概览">
    <div><span>待复核</span><strong>{{ pending }}</strong></div>
    <div><span>异常提示</span><strong class="danger">{{ exceptions }}</strong></div>
    <div><span>年度已占用</span><strong>¥ {{ occupied }}</strong></div>
    <div><span>可用余额</span><strong>¥ {{ remaining }}</strong></div>
  </section>
</template>

