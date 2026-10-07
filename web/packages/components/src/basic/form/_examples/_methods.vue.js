import { ref, shallowRef, useTemplateRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const formRef = useTemplateRef('formRef');
const message = shallowRef('');
const model = ref({
    email: '',
    nickname: '',
});
const validationSchema = {
    email(value) {
        return /^\S[^\s@]*@\S[^\s.]*\.\S+$/.test(value) ? true : '请输入有效邮箱';
    },
    nickname(value) {
        return value.length >= 2 ? true : '昵称至少 2 个字符';
    },
};
async function submit() {
    await formRef.value?.submit();
}
function handleSubmit() {
    message.value = '提交成功';
}
async function validate() {
    const result = await formRef.value?.validate();
    message.value = result?.valid ? '校验通过' : '校验未通过';
}
function fillDemo() {
    formRef.value?.setFieldValue('email', 'admin@example.com');
    formRef.value?.setFieldValue('nickname', 'Fantastic');
}
function resetFields() {
    formRef.value?.resetFields();
    message.value = '';
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
    ref: "formRef",
    model: (__VLS_ctx.model),
    validationSchema: (__VLS_ctx.validationSchema),
    scrollToError: true,
}));
const __VLS_2 = __VLS_1({
    ...{ 'onSubmit': {} },
    ref: "formRef",
    model: (__VLS_ctx.model),
    validationSchema: (__VLS_ctx.validationSchema),
    scrollToError: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.submit} */
    onSubmit: (__VLS_ctx.handleSubmit),
};
var __VLS_7;
const { default: __VLS_9 } = __VLS_3.slots;
const __VLS_10 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
    name: "email",
    label: "邮箱",
    required: true,
}));
const __VLS_12 = __VLS_11({
    name: "email",
    label: "邮箱",
    required: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_11));
const { default: __VLS_15 } = __VLS_13.slots;
const __VLS_16 = FaInput;
// @ts-ignore
const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
    type: "email",
    ...{ class: "w-full" },
}));
const __VLS_18 = __VLS_17({
    type: "email",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_17));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[model, validationSchema, handleSubmit,];
var __VLS_13;
const __VLS_21 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_22 = __VLS_asFunctionalComponent1(__VLS_21, new __VLS_21({
    name: "nickname",
    label: "昵称",
    required: true,
}));
const __VLS_23 = __VLS_22({
    name: "nickname",
    label: "昵称",
    required: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_22));
const { default: __VLS_26 } = __VLS_24.slots;
const __VLS_27 = FaInput;
// @ts-ignore
const __VLS_28 = __VLS_asFunctionalComponent1(__VLS_27, new __VLS_27({
    ...{ class: "w-full" },
}));
const __VLS_29 = __VLS_28({
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_28));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[];
var __VLS_24;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-wrap gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_32 = FaButton || FaButton;
// @ts-ignore
const __VLS_33 = __VLS_asFunctionalComponent1(__VLS_32, new __VLS_32({
    ...{ 'onClick': {} },
    type: "button",
}));
const __VLS_34 = __VLS_33({
    ...{ 'onClick': {} },
    type: "button",
}, ...__VLS_functionalComponentArgsRest(__VLS_33));
let __VLS_37;
const __VLS_38 = {
    /** @type {typeof __VLS_37.click} */
    onClick: (__VLS_ctx.submit),
};
const { default: __VLS_39 } = __VLS_35.slots;
// @ts-ignore
[submit,];
var __VLS_35;
var __VLS_36;
const __VLS_40 = FaButton || FaButton;
// @ts-ignore
const __VLS_41 = __VLS_asFunctionalComponent1(__VLS_40, new __VLS_40({
    ...{ 'onClick': {} },
    type: "button",
}));
const __VLS_42 = __VLS_41({
    ...{ 'onClick': {} },
    type: "button",
}, ...__VLS_functionalComponentArgsRest(__VLS_41));
let __VLS_45;
const __VLS_46 = {
    /** @type {typeof __VLS_45.click} */
    onClick: (__VLS_ctx.validate),
};
const { default: __VLS_47 } = __VLS_43.slots;
// @ts-ignore
[validate,];
var __VLS_43;
var __VLS_44;
const __VLS_48 = FaButton || FaButton;
// @ts-ignore
const __VLS_49 = __VLS_asFunctionalComponent1(__VLS_48, new __VLS_48({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}));
const __VLS_50 = __VLS_49({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_49));
let __VLS_53;
const __VLS_54 = {
    /** @type {typeof __VLS_53.click} */
    onClick: (__VLS_ctx.fillDemo),
};
const { default: __VLS_55 } = __VLS_51.slots;
// @ts-ignore
[fillDemo,];
var __VLS_51;
var __VLS_52;
const __VLS_56 = FaButton || FaButton;
// @ts-ignore
const __VLS_57 = __VLS_asFunctionalComponent1(__VLS_56, new __VLS_56({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}));
const __VLS_58 = __VLS_57({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_57));
let __VLS_61;
const __VLS_62 = {
    /** @type {typeof __VLS_61.click} */
    onClick: (__VLS_ctx.resetFields),
};
const { default: __VLS_63 } = __VLS_59.slots;
// @ts-ignore
[resetFields,];
var __VLS_59;
var __VLS_60;
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
if (__VLS_ctx.message) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-sm text-muted-foreground" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    (__VLS_ctx.message);
}
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[message, message,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
