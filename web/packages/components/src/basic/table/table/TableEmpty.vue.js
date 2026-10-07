import { reactiveOmit } from '@vueuse/core';
import { cn } from '#utils';
import TableCell from './TableCell.vue';
import TableRow from './TableRow.vue';
const props = withDefaults(defineProps(), {
    colspan: 1,
});
const delegatedProps = reactiveOmit(props, 'class');
const __VLS_defaults = {
    colspan: 1,
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = TableRow || TableRow;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ class: "h-full" },
}));
const __VLS_2 = __VLS_1({
    ...{ class: "h-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
/** @type {__VLS_StyleScopedClasses['h-full']} */ ;
const { default: __VLS_6 } = __VLS_3.slots;
const __VLS_7 = TableCell || TableCell;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    ...{ class: (__VLS_ctx.cn('h-full p-4 whitespace-nowrap align-middle text-sm text-foreground', props.class)) },
    ...(__VLS_ctx.delegatedProps),
}));
const __VLS_9 = __VLS_8({
    ...{ class: (__VLS_ctx.cn('h-full p-4 whitespace-nowrap align-middle text-sm text-foreground', props.class)) },
    ...(__VLS_ctx.delegatedProps),
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex h-full min-h-40 items-center justify-center" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['h-full']} */ ;
/** @type {__VLS_StyleScopedClasses['min-h-40']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
/** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
var __VLS_13 = {};
// @ts-ignore
[cn, delegatedProps,];
var __VLS_10;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_14 = __VLS_13;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
