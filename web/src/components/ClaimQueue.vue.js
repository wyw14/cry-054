const __VLS_props = defineProps();
const __VLS_emit = defineEmits();
const statusName = {
    submitted: '待分派', under_review: '复核中', need_supplement: '待补件',
    approved: '已通过', settled: '已结算', rejected: '已退回',
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.section, __VLS_intrinsics.section)({
    ...{ class: "queue" },
});
/** @type {__VLS_StyleScopedClasses['queue']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.header, __VLS_intrinsics.header)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.h2, __VLS_intrinsics.h2)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
    ...{ class: "icon-button" },
    title: "刷新",
});
/** @type {__VLS_StyleScopedClasses['icon-button']} */ ;
if (__VLS_ctx.loading) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "empty" },
    });
    /** @type {__VLS_StyleScopedClasses['empty']} */ ;
}
else if (!__VLS_ctx.claims.length) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "empty" },
    });
    /** @type {__VLS_StyleScopedClasses['empty']} */ ;
}
for (const [claim] of __VLS_vFor((__VLS_ctx.claims))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (...[$event]) => {
                return (__VLS_ctx.$emit('select', claim.id));
                // @ts-ignore
                [loading, claims, claims, $emit,];
            } },
        key: (claim.id),
        ...{ class: "queue-row" },
        ...{ class: ({ selected: claim.id === __VLS_ctx.selected }) },
    });
    /** @type {__VLS_StyleScopedClasses['queue-row']} */ ;
    /** @type {__VLS_StyleScopedClasses['selected']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "claim-title" },
    });
    /** @type {__VLS_StyleScopedClasses['claim-title']} */ ;
    (claim.category);
    (claim.id);
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "money" },
    });
    /** @type {__VLS_StyleScopedClasses['money']} */ ;
    (claim.amount);
    __VLS_asFunctionalElement1(__VLS_intrinsics.small, __VLS_intrinsics.small)({});
    (claim.occurred_on.slice(0, 10));
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "status" },
    });
    /** @type {__VLS_StyleScopedClasses['status']} */ ;
    (__VLS_ctx.statusName[claim.status] ?? claim.status);
    // @ts-ignore
    [selected, statusName,];
}
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
export default {};
