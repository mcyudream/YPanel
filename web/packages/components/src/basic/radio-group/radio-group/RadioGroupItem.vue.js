import { CircleIcon } from '@lucide/vue';
import { reactiveOmit } from '@vueuse/core';
import { RadioGroupIndicator, RadioGroupItem, useForwardProps, } from 'reka-ui';
import { cn } from '#utils';
const props = defineProps();
const delegatedProps = reactiveOmit(props, 'class');
const forwardedProps = useForwardProps(delegatedProps);
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
/** @ts-ignore @type { | typeof __VLS_components.RadioGroupItem | typeof __VLS_components.RadioGroupItem} */
RadioGroupItem;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "radio-group-item",
    ...(__VLS_ctx.forwardedProps),
    ...{ class: (__VLS_ctx.cn('border-input text-primary focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:bg-input/30 aspect-square size-4 shrink-0 rounded-full border shadow-xs transition-[color,box-shadow] outline-none focus-visible:ring-3 disabled:cursor-not-allowed disabled:opacity-50', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "radio-group-item",
    ...(__VLS_ctx.forwardedProps),
    ...{ class: (__VLS_ctx.cn('border-input text-primary focus-visible:border-ring focus-visible:ring-ring/50 aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive dark:bg-input/30 aspect-square size-4 shrink-0 rounded-full border shadow-xs transition-[color,box-shadow] outline-none focus-visible:ring-3 disabled:cursor-not-allowed disabled:opacity-50', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.RadioGroupIndicator | typeof __VLS_components.RadioGroupIndicator} */
RadioGroupIndicator;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    dataSlot: "radio-group-indicator",
    ...{ class: "flex items-center justify-center relative" },
}));
const __VLS_9 = __VLS_8({
    dataSlot: "radio-group-indicator",
    ...{ class: "flex items-center justify-center relative" },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
/** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
/** @type {__VLS_StyleScopedClasses['relative']} */ ;
const { default: __VLS_12 } = __VLS_10.slots;
var __VLS_13 = {};
let __VLS_15;
/** @ts-ignore @type { | typeof __VLS_components.CircleIcon} */
CircleIcon;
// @ts-ignore
const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({
    ...{ class: "size-2 left-1/2 top-1/2 absolute fill-primary -translate-x-1/2 -translate-y-1/2" },
}));
const __VLS_17 = __VLS_16({
    ...{ class: "size-2 left-1/2 top-1/2 absolute fill-primary -translate-x-1/2 -translate-y-1/2" },
}, ...__VLS_functionalComponentArgsRest(__VLS_16));
/** @type {__VLS_StyleScopedClasses['size-2']} */ ;
/** @type {__VLS_StyleScopedClasses['left-1/2']} */ ;
/** @type {__VLS_StyleScopedClasses['top-1/2']} */ ;
/** @type {__VLS_StyleScopedClasses['absolute']} */ ;
/** @type {__VLS_StyleScopedClasses['fill-primary']} */ ;
/** @type {__VLS_StyleScopedClasses['-translate-x-1/2']} */ ;
/** @type {__VLS_StyleScopedClasses['-translate-y-1/2']} */ ;
// @ts-ignore
[forwardedProps, cn,];
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
});
const __VLS_export = {};
export default {};
