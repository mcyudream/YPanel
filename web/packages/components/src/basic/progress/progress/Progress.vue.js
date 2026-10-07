import { reactiveOmit } from '@vueuse/core';
import { ProgressIndicator, ProgressRoot, } from 'reka-ui';
import { cn } from '#utils';
const props = withDefaults(defineProps(), {
    modelValue: 0,
});
const delegatedProps = reactiveOmit(props, 'class');
const __VLS_defaults = {
    modelValue: 0,
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.ProgressRoot | typeof __VLS_components.ProgressRoot} */
ProgressRoot;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "progress",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('bg-primary/20 relative h-2 w-full overflow-hidden rounded-full', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "progress",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('bg-primary/20 relative h-2 w-full overflow-hidden rounded-full', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.ProgressIndicator} */
ProgressIndicator;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    dataSlot: "progress-indicator",
    ...{ class: "bg-primary flex-1 h-full w-full transition-all" },
    ...{ style: (`transform: translateX(-${100 - (props.modelValue ?? 0)}%);`) },
}));
const __VLS_9 = __VLS_8({
    dataSlot: "progress-indicator",
    ...{ class: "bg-primary flex-1 h-full w-full transition-all" },
    ...{ style: (`transform: translateX(-${100 - (props.modelValue ?? 0)}%);`) },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
/** @type {__VLS_StyleScopedClasses['bg-primary']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
/** @type {__VLS_StyleScopedClasses['h-full']} */ ;
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
/** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
// @ts-ignore
[delegatedProps, cn,];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
export default {};
