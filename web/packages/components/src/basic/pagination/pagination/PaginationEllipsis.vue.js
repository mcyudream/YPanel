import { MoreHorizontal } from '@lucide/vue';
import { reactiveOmit } from '@vueuse/core';
import { PaginationEllipsis } from 'reka-ui';
import { cn } from '#utils';
const props = defineProps();
const delegatedProps = reactiveOmit(props, 'class');
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.PaginationEllipsis | typeof __VLS_components.PaginationEllipsis} */
PaginationEllipsis;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "pagination-ellipsis",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('flex size-9 items-center justify-center', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "pagination-ellipsis",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('flex size-9 items-center justify-center', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
var __VLS_7 = {};
let __VLS_9;
/** @ts-ignore @type { | typeof __VLS_components.MoreHorizontal} */
MoreHorizontal;
// @ts-ignore
const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
    ...{ class: "size-4" },
}));
const __VLS_11 = __VLS_10({
    ...{ class: "size-4" },
}, ...__VLS_functionalComponentArgsRest(__VLS_10));
/** @type {__VLS_StyleScopedClasses['size-4']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
    ...{ class: "sr-only" },
});
/** @type {__VLS_StyleScopedClasses['sr-only']} */ ;
// @ts-ignore
[delegatedProps, cn,];
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
