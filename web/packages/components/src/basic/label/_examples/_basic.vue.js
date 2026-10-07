import { reactive } from 'vue';
import FaInput from '../../input/index.vue';
import FaLabel from '../index.vue';
const form = reactive({
    name: '',
    email: '',
});
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "space-y-4" },
});
/** @type {__VLS_StyleScopedClasses['space-y-4']} */ ;
const __VLS_0 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    label: "用户名",
}));
const __VLS_2 = __VLS_1({
    label: "用户名",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
const __VLS_6 = FaInput;
// @ts-ignore
const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({
    modelValue: (__VLS_ctx.form.name),
    placeholder: "请输入用户名",
    ...{ class: "w-72" },
}));
const __VLS_8 = __VLS_7({
    modelValue: (__VLS_ctx.form.name),
    placeholder: "请输入用户名",
    ...{ class: "w-72" },
}, ...__VLS_functionalComponentArgsRest(__VLS_7));
/** @type {__VLS_StyleScopedClasses['w-72']} */ ;
// @ts-ignore
[form,];
var __VLS_3;
const __VLS_11 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_12 = __VLS_asFunctionalComponent1(__VLS_11, new __VLS_11({
    label: "邮箱",
}));
const __VLS_13 = __VLS_12({
    label: "邮箱",
}, ...__VLS_functionalComponentArgsRest(__VLS_12));
const { default: __VLS_16 } = __VLS_14.slots;
const __VLS_17 = FaInput;
// @ts-ignore
const __VLS_18 = __VLS_asFunctionalComponent1(__VLS_17, new __VLS_17({
    modelValue: (__VLS_ctx.form.email),
    placeholder: "请输入邮箱",
    ...{ class: "w-72" },
}));
const __VLS_19 = __VLS_18({
    modelValue: (__VLS_ctx.form.email),
    placeholder: "请输入邮箱",
    ...{ class: "w-72" },
}, ...__VLS_functionalComponentArgsRest(__VLS_18));
/** @type {__VLS_StyleScopedClasses['w-72']} */ ;
// @ts-ignore
[form,];
var __VLS_14;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
