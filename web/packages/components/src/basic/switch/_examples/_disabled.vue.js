import { ref } from 'vue';
import FaSwitch from '../index.vue';
const off = ref(false);
const on = ref(true);
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex gap-4" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
const __VLS_0 = FaSwitch;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.off),
    disabled: true,
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.off),
    disabled: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = FaSwitch;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    modelValue: (__VLS_ctx.on),
    disabled: true,
}));
const __VLS_7 = __VLS_6({
    modelValue: (__VLS_ctx.on),
    disabled: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
// @ts-ignore
[off, on,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
