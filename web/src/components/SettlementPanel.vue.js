import { computed, ref } from 'vue';
import { api } from '../services/api';
const props = defineProps();
const preview = ref(null);
const busy = ref(false);
const message = ref('');
const claimTitle = computed(() => props.claim ? `${props.claim.category} · ${props.claim.id}` : '请选择申报');
async function runPreview() {
    if (!props.claim)
        return;
    busy.value = true;
    message.value = '';
    try {
        preview.value = await api(`/claims/${props.claim.id}/preview`, { method: 'POST' });
    }
    catch (cause) {
        message.value = cause instanceof Error ? cause.message : '试算失败';
    }
    finally {
        busy.value = false;
    }
}
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.section, __VLS_intrinsics.section)({
    ...{ class: "settlement" },
});
/** @type {__VLS_StyleScopedClasses['settlement']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.header, __VLS_intrinsics.header)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
    ...{ class: "eyebrow" },
});
/** @type {__VLS_StyleScopedClasses['eyebrow']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.h2, __VLS_intrinsics.h2)({});
(__VLS_ctx.claimTitle);
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
    ...{ onClick: (__VLS_ctx.runPreview) },
    disabled: (!__VLS_ctx.claim || __VLS_ctx.busy),
});
(__VLS_ctx.busy ? '计算中' : '重新试算');
if (!__VLS_ctx.claim) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "empty" },
    });
    /** @type {__VLS_StyleScopedClasses['empty']} */ ;
}
else {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "summary-grid" },
    });
    /** @type {__VLS_StyleScopedClasses['summary-grid']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.strong, __VLS_intrinsics.strong)({});
    (__VLS_ctx.claim.amount);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.strong, __VLS_intrinsics.strong)({});
    (__VLS_ctx.claim.status);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.strong, __VLS_intrinsics.strong)({});
    (__VLS_ctx.claim.version);
    if (__VLS_ctx.message) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
            ...{ class: "inline-error" },
        });
        /** @type {__VLS_StyleScopedClasses['inline-error']} */ ;
        (__VLS_ctx.message);
    }
    if (__VLS_ctx.preview) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.pre, __VLS_intrinsics.pre)({
            ...{ class: "preview-json" },
        });
        /** @type {__VLS_StyleScopedClasses['preview-json']} */ ;
        (JSON.stringify(__VLS_ctx.preview, null, 2));
    }
    else {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "explanation-placeholder" },
        });
        /** @type {__VLS_StyleScopedClasses['explanation-placeholder']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    }
}
// @ts-ignore
[claimTitle, runPreview, claim, claim, claim, claim, claim, busy, busy, message, message, preview, preview,];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
});
export default {};
