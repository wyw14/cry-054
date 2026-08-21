import { onMounted } from 'vue';
import ClaimQueue from './components/ClaimQueue.vue';
import MetricStrip from './components/MetricStrip.vue';
import SettlementPanel from './components/SettlementPanel.vue';
import { useWorkspaceStore } from './stores/workspace';
const store = useWorkspaceStore();
const navigation = [
    ['workbench', '处理工作台'], ['rules', '规则中心'], ['claims', '申报录入'],
    ['ledger', '年度台账'], ['compare', '差异对比'], ['exports', '受控导出'], ['audit', '审计记录'],
];
onMounted(() => store.loadClaims());
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "shell" },
});
/** @type {__VLS_StyleScopedClasses['shell']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.aside, __VLS_intrinsics.aside)({
    ...{ class: "sidebar" },
});
/** @type {__VLS_StyleScopedClasses['sidebar']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "brand" },
});
/** @type {__VLS_StyleScopedClasses['brand']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
    ...{ class: "brand-mark" },
});
/** @type {__VLS_StyleScopedClasses['brand-mark']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.strong, __VLS_intrinsics.strong)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.small, __VLS_intrinsics.small)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.nav, __VLS_intrinsics.nav)({});
for (const [item] of __VLS_vFor((__VLS_ctx.navigation))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (...[$event]) => {
                return (__VLS_ctx.store.selectView(item[0]));
                // @ts-ignore
                [navigation, store,];
            } },
        key: (item[0]),
        ...{ class: ({ active: __VLS_ctx.store.activeView === item[0] }) },
    });
    /** @type {__VLS_StyleScopedClasses['active']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "nav-dot" },
    });
    /** @type {__VLS_StyleScopedClasses['nav-dot']} */ ;
    (item[1]);
    // @ts-ignore
    [store,];
}
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "sidebar-footer" },
});
/** @type {__VLS_StyleScopedClasses['sidebar-footer']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
    ...{ class: "health-dot" },
});
/** @type {__VLS_StyleScopedClasses['health-dot']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.strong, __VLS_intrinsics.strong)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.small, __VLS_intrinsics.small)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.main, __VLS_intrinsics.main)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.header, __VLS_intrinsics.header)({
    ...{ class: "topbar" },
});
/** @type {__VLS_StyleScopedClasses['topbar']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
    ...{ class: "eyebrow" },
});
/** @type {__VLS_StyleScopedClasses['eyebrow']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.h1, __VLS_intrinsics.h1)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "operator" },
});
/** @type {__VLS_StyleScopedClasses['operator']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.strong, __VLS_intrinsics.strong)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.small, __VLS_intrinsics.small)({});
const __VLS_0 = MetricStrip;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    pending: (__VLS_ctx.store.total),
    exceptions: (0),
    occupied: "0.00",
    remaining: "50000.00",
}));
const __VLS_2 = __VLS_1({
    pending: (__VLS_ctx.store.total),
    exceptions: (0),
    occupied: "0.00",
    remaining: "50000.00",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
if (__VLS_ctx.store.error) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
        ...{ class: "page-error" },
    });
    /** @type {__VLS_StyleScopedClasses['page-error']} */ ;
    (__VLS_ctx.store.error);
}
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "workspace-grid" },
});
/** @type {__VLS_StyleScopedClasses['workspace-grid']} */ ;
const __VLS_5 = ClaimQueue;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    ...{ 'onSelect': {} },
    claims: (__VLS_ctx.store.claims),
    selected: (__VLS_ctx.store.selectedClaimId),
    loading: (__VLS_ctx.store.loading),
}));
const __VLS_7 = __VLS_6({
    ...{ 'onSelect': {} },
    claims: (__VLS_ctx.store.claims),
    selected: (__VLS_ctx.store.selectedClaimId),
    loading: (__VLS_ctx.store.loading),
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
let __VLS_10;
const __VLS_11 = {
    /** @type {typeof __VLS_10.select} */
    onSelect: (...[$event]) => {
        return (__VLS_ctx.store.selectedClaimId = $event);
        // @ts-ignore
        [store, store, store, store, store, store, store,];
    },
};
var __VLS_8;
var __VLS_9;
const __VLS_12 = SettlementPanel;
// @ts-ignore
const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
    claim: (__VLS_ctx.store.selectedClaim),
}));
const __VLS_14 = __VLS_13({
    claim: (__VLS_ctx.store.selectedClaim),
}, ...__VLS_functionalComponentArgsRest(__VLS_13));
// @ts-ignore
[store,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
