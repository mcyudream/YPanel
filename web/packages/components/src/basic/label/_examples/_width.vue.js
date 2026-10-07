import { reactive } from 'vue';
import FaInput from '../../input/index.vue';
import FaSelect from '../../select/index.vue';
import FaLabel from '../index.vue';
const form = reactive({
    name: '',
    role: 'admin',
    phone: '',
});
const roleOptions = [
    { label: '管理员', value: 'admin' },
    { label: '运营', value: 'operator' },
    { label: '访客', value: 'guest' },
];
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
    labelWidth: (88),
}));
const __VLS_2 = __VLS_1({
    label: "用户名",
    labelWidth: (88),
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
    label: "角色",
    labelWidth: (88),
}));
const __VLS_13 = __VLS_12({
    label: "角色",
    labelWidth: (88),
}, ...__VLS_functionalComponentArgsRest(__VLS_12));
const { default: __VLS_16 } = __VLS_14.slots;
const __VLS_17 = FaSelect;
// @ts-ignore
const __VLS_18 = __VLS_asFunctionalComponent1(__VLS_17, new __VLS_17({
    modelValue: (__VLS_ctx.form.role),
    options: (__VLS_ctx.roleOptions),
    ...{ class: "w-72" },
}));
const __VLS_19 = __VLS_18({
    modelValue: (__VLS_ctx.form.role),
    options: (__VLS_ctx.roleOptions),
    ...{ class: "w-72" },
}, ...__VLS_functionalComponentArgsRest(__VLS_18));
/** @type {__VLS_StyleScopedClasses['w-72']} */ ;
// @ts-ignore
[form, roleOptions,];
var __VLS_14;
const __VLS_22 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_23 = __VLS_asFunctionalComponent1(__VLS_22, new __VLS_22({
    label: "手机号",
    labelWidth: (88),
}));
const __VLS_24 = __VLS_23({
    label: "手机号",
    labelWidth: (88),
}, ...__VLS_functionalComponentArgsRest(__VLS_23));
const { default: __VLS_27 } = __VLS_25.slots;
const __VLS_28 = FaInput;
// @ts-ignore
const __VLS_29 = __VLS_asFunctionalComponent1(__VLS_28, new __VLS_28({
    modelValue: (__VLS_ctx.form.phone),
    placeholder: "请输入手机号",
    ...{ class: "w-72" },
}));
const __VLS_30 = __VLS_29({
    modelValue: (__VLS_ctx.form.phone),
    placeholder: "请输入手机号",
    ...{ class: "w-72" },
}, ...__VLS_functionalComponentArgsRest(__VLS_29));
/** @type {__VLS_StyleScopedClasses['w-72']} */ ;
// @ts-ignore
[form,];
var __VLS_25;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
