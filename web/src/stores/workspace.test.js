import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useWorkspaceStore } from './workspace';
describe('workspace store', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        vi.restoreAllMocks();
    });
    it('loads the review queue and selects the first claim', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({
            items: [{
                    id: 'claim-1', claimant_id: 'person-1', project_id: 'project-1',
                    category: 'medical', occurred_on: '2026-05-10T00:00:00Z',
                    amount: '1200.00', status: 'submitted', version: 1,
                }],
            page: 1,
            page_size: 50,
            total: 1,
        }), { status: 200, headers: { 'Content-Type': 'application/json' } })));
        const store = useWorkspaceStore();
        await store.loadClaims('submitted');
        expect(store.total).toBe(1);
        expect(store.selectedClaimId).toBe('claim-1');
        expect(store.selectedClaim?.amount).toBe('1200.00');
        expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/api/v1/claims?'), expect.objectContaining({ headers: expect.any(Headers) }));
    });
    it('keeps a readable error when the API rejects the queue request', async () => {
        vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({
            code: 'NOT_READY', message: '服务尚未就绪', request_id: 'req-test',
        }), { status: 503, headers: { 'Content-Type': 'application/json' } })));
        const store = useWorkspaceStore();
        await store.loadClaims();
        expect(store.error).toBe('服务尚未就绪');
        expect(store.loading).toBe(false);
        expect(store.claims).toEqual([]);
    });
});
