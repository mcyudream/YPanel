import { ref } from 'vue';
import FaButton from '../../button/index.vue';
import FaIcon from '../../icon/index.vue';
import FaTooltip from '../../tooltip/index.vue';
import FaInput from '../index.vue';
const value = ref('');
const value2 = ref('');
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-col gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-col']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_0 = FaInput || FaInput;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.value),
    placeholder: "example.com",
    inputClass: "ps-1",
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.value),
    placeholder: "example.com",
    inputClass: "ps-1",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
{
    const { start: __VLS_6 } = __VLS_3.slots;
    // @ts-ignore
    [value,];
}
{
    const { end: __VLS_7 } = __VLS_3.slots;
    const __VLS_8 = FaTooltip || FaTooltip;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        text: "可输入域名、IP、端口、URL 等",
    }));
    const __VLS_10 = __VLS_9({
        text: "可输入域名、IP、端口、URL 等",
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    const { default: __VLS_13 } = __VLS_11.slots;
    const __VLS_14 = FaIcon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        name: "i-ri:question-line",
        ...{ class: "text-base text-orange cursor-help" },
    }));
    const __VLS_16 = __VLS_15({
        name: "i-ri:question-line",
        ...{ class: "text-base text-orange cursor-help" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    /** @type {__VLS_StyleScopedClasses['text-base']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-orange']} */ ;
    /** @type {__VLS_StyleScopedClasses['cursor-help']} */ ;
    // @ts-ignore
    [];
    var __VLS_11;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
const __VLS_19 = FaInput || FaInput;
// @ts-ignore
const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
    modelValue: (__VLS_ctx.value2),
    placeholder: "请输入内容",
    align: "block",
    inputClass: "shadow-none",
    endClass: "justify-end",
}));
const __VLS_21 = __VLS_20({
    modelValue: (__VLS_ctx.value2),
    placeholder: "请输入内容",
    align: "block",
    inputClass: "shadow-none",
    endClass: "justify-end",
}, ...__VLS_functionalComponentArgsRest(__VLS_20));
const { default: __VLS_24 } = __VLS_22.slots;
{
    const { start: __VLS_25 } = __VLS_22.slots;
    // @ts-ignore
    [value2,];
}
{
    const { end: __VLS_26 } = __VLS_22.slots;
    const __VLS_27 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_28 = __VLS_asFunctionalComponent1(__VLS_27, new __VLS_27({
        variant: "ghost",
        size: "sm",
        ...{ class: "px-2 h-8" },
    }));
    const __VLS_29 = __VLS_28({
        variant: "ghost",
        size: "sm",
        ...{ class: "px-2 h-8" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_28));
    /** @type {__VLS_StyleScopedClasses['px-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['h-8']} */ ;
    const { default: __VLS_32 } = __VLS_30.slots;
    // @ts-ignore
    [];
    var __VLS_30;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_22;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
