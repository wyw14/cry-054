import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { api } from '../services/api';
export const useWorkspaceStore = defineStore('workspace', () => {
    const claims = ref([]);
    const selectedClaimId = ref('');
    const loading = ref(false);
    const error = ref('');
    const activeView = ref('workbench');
    const total = ref(0);
    const selectedClaim = computed(() => claims.value.find(item => item.id === selectedClaimId.value));
    async function loadClaims(status = '') {
        loading.value = true;
        error.value = '';
        try {
            const query = new URLSearchParams({ page: '1', page_size: '50', sort_by: 'created_at', sort_order: 'desc' });
            if (status)
                query.set('status', status);
            const page = await api(`/claims?${query}`);
            claims.value = page.items;
            total.value = page.total;
            if (!selectedClaimId.value && page.items.length)
                selectedClaimId.value = page.items[0].id;
        }
        catch (cause) {
            error.value = cause instanceof Error ? cause.message : '加载失败';
        }
        finally {
            loading.value = false;
        }
    }
    function selectView(view) {
        activeView.value = view;
    }
    return { claims, selectedClaimId, selectedClaim, activeView, total, loading, error, loadClaims, selectView };
});
