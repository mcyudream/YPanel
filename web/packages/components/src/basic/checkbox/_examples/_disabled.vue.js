import { shallowRef } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaCheckbox from '../index.vue';
const unchecked = shallowRef(false);
const checked = shallowRef(true);
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-col gap-3" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-col']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-3']} */ ;
const __VLS_0 = FaCheckbox || FaCheckbox;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.unchecked),
    disabled: true,
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.unchecked),
    disabled: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
// @ts-ignore
[unchecked,];
var __VLS_3;
const __VLS_6 = FaCheckbox || FaCheckbox;
// @ts-ignore
const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({
    modelValue: (__VLS_ctx.checked),
    disabled: true,
}));
const __VLS_8 = __VLS_7({
    modelValue: (__VLS_ctx.checked),
    disabled: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_7));
const { default: __VLS_11 } = __VLS_9.slots;
// @ts-ignore
[checked,];
var __VLS_9;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
