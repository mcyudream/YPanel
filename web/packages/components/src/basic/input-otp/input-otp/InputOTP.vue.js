import { reactiveOmit } from '@vueuse/core';
import { useForwardPropsEmits } from 'reka-ui';
import { OTPInput } from 'vue-input-otp';
import { cn } from '#utils';
const props = defineProps();
const emits = defineEmits();
const delegatedProps = reactiveOmit(props, 'class');
const forwarded = useForwardPropsEmits(delegatedProps, emits);
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.OTPInput | typeof __VLS_components.OTPInput} */
OTPInput;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...(__VLS_ctx.forwarded),
    containerClass: (__VLS_ctx.cn('flex items-center gap-2 has-disabled:opacity-50', props.class)),
    dataSlot: "input-otp",
    ...{ class: "disabled:cursor-not-allowed" },
}));
const __VLS_2 = __VLS_1({
    ...(__VLS_ctx.forwarded),
    containerClass: (__VLS_ctx.cn('flex items-center gap-2 has-disabled:opacity-50', props.class)),
    dataSlot: "input-otp",
    ...{ class: "disabled:cursor-not-allowed" },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
/** @type {__VLS_StyleScopedClasses['disabled:cursor-not-allowed']} */ ;
{
    const { default: __VLS_6 } = __VLS_3.slots;
    const [slotProps] = __VLS_vSlot(__VLS_6);
    var __VLS_7 = {
        ...(slotProps),
    };
    // @ts-ignore
    [forwarded, cn,];
    __VLS_3.slots['' /* empty slot name completion */];
}
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
