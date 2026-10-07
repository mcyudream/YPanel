import { reactiveOmit } from '@vueuse/core';
import { SelectSeparator } from 'reka-ui';
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
/** @ts-ignore @type { | typeof __VLS_components.SelectSeparator} */
SelectSeparator;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "select-separator",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('bg-border pointer-events-none -mx-1 my-1 h-px', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "select-separator",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('bg-border pointer-events-none -mx-1 my-1 h-px', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[delegatedProps, cn,];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
});
export default {};
