<script setup lang="ts">
import { onMounted } from 'vue'
import ClaimQueue from './components/ClaimQueue.vue'
import MetricStrip from './components/MetricStrip.vue'
import SettlementPanel from './components/SettlementPanel.vue'
import { useWorkspaceStore } from './stores/workspace'

const store = useWorkspaceStore()
const navigation = [
  ['workbench', '处理工作台'], ['rules', '规则中心'], ['claims', '申报录入'],
  ['ledger', '年度台账'], ['compare', '差异对比'], ['exports', '受控导出'], ['audit', '审计记录'],
]

onMounted(() => store.loadClaims())
</script>

<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand"><span class="brand-mark">益</span><div><strong>公益补助结算</strong><small>GrantLedger</small></div></div>
      <nav>
        <button v-for="item in navigation" :key="item[0]" :class="{ active: store.activeView === item[0] }" @click="store.selectView(item[0])">
          <span class="nav-dot"></span>{{ item[1] }}
        </button>
      </nav>
      <div class="sidebar-footer"><span class="health-dot"></span><div><strong>本地服务正常</strong><small>离线适配器已启用</small></div></div>
    </aside>
    <main>
      <header class="topbar">
        <div><p class="eyebrow">2026 年度 · 困难家庭医疗补助</p><h1>管理员处理工作台</h1></div>
        <div class="operator"><span>管</span><div><strong>演示管理员</strong><small>规则与复核权限</small></div></div>
      </header>
      <MetricStrip :pending="store.total" :exceptions="0" occupied="0.00" remaining="50000.00" />
      <p v-if="store.error" class="page-error">{{ store.error }}</p>
      <div class="workspace-grid">
        <ClaimQueue :claims="store.claims" :selected="store.selectedClaimId" :loading="store.loading" @select="store.selectedClaimId = $event" />
        <SettlementPanel :claim="store.selectedClaim" />
      </div>
    </main>
  </div>
</template>

