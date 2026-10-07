import { ChevronDown } from '@lucide/vue';
import { reactiveOmit } from '@vueuse/core';
import { SelectScrollDownButton, useForwardProps } from 'reka-ui';
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
/** @ts-ignore @type { | typeof __VLS_components.SelectScrollDownButton | typeof __VLS_components.SelectScrollDownButton} */
SelectScrollDownButton;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "select-scroll-down-button",
    ...(__VLS_ctx.forwardedProps),
    ...{ class: (__VLS_ctx.cn('flex cursor-default items-center justify-center py-1', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "select-scroll-down-button",
    ...(__VLS_ctx.forwardedProps),
    ...{ class: (__VLS_ctx.cn('flex cursor-default items-center justify-center py-1', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
var __VLS_7 = {};
let __VLS_9;
/** @ts-ignore @type { | typeof __VLS_components.ChevronDown} */
ChevronDown;
// @ts-ignore
const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
    ...{ class: "size-4" },
}));
const __VLS_11 = __VLS_10({
    ...{ class: "size-4" },
}, ...__VLS_functionalComponentArgsRest(__VLS_10));
/** @type {__VLS_StyleScopedClasses['size-4']} */ ;
// @ts-ignore
[forwardedProps, cn,];
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
