import { reactiveOmit } from '@vueuse/core';
import { DialogDescription, useForwardProps } from 'reka-ui';
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
/** @ts-ignore @type { | typeof __VLS_components.DialogDescription | typeof __VLS_components.DialogDescription} */
DialogDescription;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "dialog-description",
    ...(__VLS_ctx.forwardedProps),
    ...{ class: (__VLS_ctx.cn('text-muted-foreground text-sm', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "dialog-description",
    ...(__VLS_ctx.forwardedProps),
    ...{ class: (__VLS_ctx.cn('text-muted-foreground text-sm', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
var __VLS_7 = {};
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
