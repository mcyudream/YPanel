import { ref, shallowRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaSelect from '../../select/index.vue';
import FaTextarea from '../../textarea/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const submitted = shallowRef('');
const model = ref({
    account: '',
    role: '',
    remark: '',
});
const roleOptions = [
    { label: '管理员', value: 'admin' },
    { label: '运营', value: 'operator' },
    { label: '访客', value: 'guest' },
];
const validationSchema = {
    account(value) {
        return value ? true : '请输入账号';
    },
    role(value) {
        return value ? true : '请选择角色';
    },
    remark(value) {
        return !value || value.length >= 6 ? true : '备注至少 6 个字符';
    },
};
function handleSubmit(values) {
    submitted.value = JSON.stringify(values, null, 2);
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-4 grid max-w-160" },
});
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
/** @type {__VLS_StyleScopedClasses['max-w-160']} */ ;
const __VLS_0 = FaForm || FaForm;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onSubmit': {} },
    model: (__VLS_ctx.model),
    validationSchema: (__VLS_ctx.validationSchema),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onSubmit': {} },
    model: (__VLS_ctx.model),
    validationSchema: (__VLS_ctx.validationSchema),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.submit} */
    onSubmit: (__VLS_ctx.handleSubmit),
};
const { default: __VLS_7 } = __VLS_3.slots;
const __VLS_8 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
    name: "account",
    label: "账号",
    required: true,
    description: "账号将作为登录名使用",
}));
const __VLS_10 = __VLS_9({
    name: "account",
    label: "账号",
    required: true,
    description: "账号将作为登录名使用",
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
const { default: __VLS_13 } = __VLS_11.slots;
const __VLS_14 = FaInput;
// @ts-ignore
const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
    placeholder: "请输入账号",
    ...{ class: "w-full" },
}));
const __VLS_16 = __VLS_15({
    placeholder: "请输入账号",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_15));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[model, validationSchema, handleSubmit,];
var __VLS_11;
const __VLS_19 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
    name: "role",
    label: "角色",
    required: true,
}));
const __VLS_21 = __VLS_20({
    name: "role",
    label: "角色",
    required: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_20));
const { default: __VLS_24 } = __VLS_22.slots;
const __VLS_25 = FaSelect;
// @ts-ignore
const __VLS_26 = __VLS_asFunctionalComponent1(__VLS_25, new __VLS_25({
    options: (__VLS_ctx.roleOptions),
    placeholder: "请选择角色",
    ...{ class: "w-full" },
}));
const __VLS_27 = __VLS_26({
    options: (__VLS_ctx.roleOptions),
    placeholder: "请选择角色",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_26));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[roleOptions,];
var __VLS_22;
const __VLS_30 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({
    name: "remark",
    label: "备注",
}));
const __VLS_32 = __VLS_31({
    name: "remark",
    label: "备注",
}, ...__VLS_functionalComponentArgsRest(__VLS_31));
const { default: __VLS_35 } = __VLS_33.slots;
const __VLS_36 = FaTextarea;
// @ts-ignore
const __VLS_37 = __VLS_asFunctionalComponent1(__VLS_36, new __VLS_36({
    placeholder: "请输入至少 6 个字符",
    ...{ class: "w-full" },
}));
const __VLS_38 = __VLS_37({
    placeholder: "请输入至少 6 个字符",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_37));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[];
var __VLS_33;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_41 = FaButton || FaButton;
// @ts-ignore
const __VLS_42 = __VLS_asFunctionalComponent1(__VLS_41, new __VLS_41({
    type: "submit",
}));
const __VLS_43 = __VLS_42({
    type: "submit",
}, ...__VLS_functionalComponentArgsRest(__VLS_42));
const { default: __VLS_46 } = __VLS_44.slots;
// @ts-ignore
[];
var __VLS_44;
const __VLS_47 = FaButton || FaButton;
// @ts-ignore
const __VLS_48 = __VLS_asFunctionalComponent1(__VLS_47, new __VLS_47({
    type: "reset",
    variant: "outline",
}));
const __VLS_49 = __VLS_48({
    type: "reset",
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_48));
const { default: __VLS_52 } = __VLS_50.slots;
// @ts-ignore
[];
var __VLS_50;
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
if (__VLS_ctx.submitted) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.pre, __VLS_intrinsics.pre)({
        ...{ class: "text-sm m-0 p-4 rounded-md bg-muted" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['m-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['p-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-muted']} */ ;
    (__VLS_ctx.submitted);
}
// @ts-ignore
[submitted, submitted,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
