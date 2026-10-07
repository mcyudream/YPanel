import { ref } from 'vue';
import FaInput from '../../input/index.vue';
import FaPasswordStrength from '../index.vue';
const password = ref('');
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex-col w-80" },
});
/** @type {__VLS_StyleScopedClasses['flex-col']} */ ;
/** @type {__VLS_StyleScopedClasses['w-80']} */ ;
const __VLS_0 = FaInput;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.password),
    placeholder: "请输入密码",
    ...{ class: "w-full" },
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.password),
    placeholder: "请输入密码",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
const __VLS_5 = FaPasswordStrength;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    password: (__VLS_ctx.password),
    ...{ class: "mt-2" },
}));
const __VLS_7 = __VLS_6({
    password: (__VLS_ctx.password),
    ...{ class: "mt-2" },
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
/** @type {__VLS_StyleScopedClasses['mt-2']} */ ;
// @ts-ignore
[password, password,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
