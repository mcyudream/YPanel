import { reactiveOmit } from '@vueuse/core';
import { cn } from '#utils';
import { Separator } from '../separator';
const props = withDefaults(defineProps(), {
    orientation: 'vertical',
});
const delegatedProps = reactiveOmit(props, 'class');
const __VLS_defaults = {
    orientation: 'vertical',
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
/** @ts-ignore @type { | typeof __VLS_components.Separator} */
Separator;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "button-group-separator",
    ...(__VLS_ctx.delegatedProps),
    orientation: (props.orientation),
    ...{ class: (__VLS_ctx.cn('bg-input relative !m-0 self-stretch data-[orientation=vertical]:h-auto', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "button-group-separator",
    ...(__VLS_ctx.delegatedProps),
    orientation: (props.orientation),
    ...{ class: (__VLS_ctx.cn('bg-input relative !m-0 self-stretch data-[orientation=vertical]:h-auto', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[delegatedProps, cn,];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
export default {};
