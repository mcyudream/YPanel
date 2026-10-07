import { shallowRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaIcon from '../../icon/index.vue';
import FaTextarea from '../index.vue';
const value = shallowRef('');
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaTextarea || FaTextarea;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.value),
    placeholder: "console.log('Hello, world!');",
    align: "block",
    startClass: "justify-between",
    endClass: "justify-between",
    ...{ class: "w-120" },
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.value),
    placeholder: "console.log('Hello, world!');",
    align: "block",
    startClass: "justify-between",
    endClass: "justify-between",
    ...{ class: "w-120" },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
/** @type {__VLS_StyleScopedClasses['w-120']} */ ;
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { start: __VLS_7 } = __VLS_3.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    const __VLS_8 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        variant: "ghost",
        size: "icon",
        ...{ class: "size-6" },
    }));
    const __VLS_10 = __VLS_9({
        variant: "ghost",
        size: "icon",
        ...{ class: "size-6" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
    const { default: __VLS_13 } = __VLS_11.slots;
    const __VLS_14 = FaIcon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        name: "i-ep:refresh",
    }));
    const __VLS_16 = __VLS_15({
        name: "i-ep:refresh",
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    // @ts-ignore
    [value,];
    var __VLS_11;
    // @ts-ignore
    [];
}
{
    const { end: __VLS_19 } = __VLS_3.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    const __VLS_20 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({
        size: "sm",
        ...{ class: "px-2 h-8" },
    }));
    const __VLS_22 = __VLS_21({
        size: "sm",
        ...{ class: "px-2 h-8" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_21));
    /** @type {__VLS_StyleScopedClasses['px-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['h-8']} */ ;
    const { default: __VLS_25 } = __VLS_23.slots;
    const __VLS_26 = FaIcon;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        name: "i-lucide:corner-down-left",
    }));
    const __VLS_28 = __VLS_27({
        name: "i-lucide:corner-down-left",
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    // @ts-ignore
    [];
    var __VLS_23;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
