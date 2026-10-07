import { ref, shallowRef, useTemplateRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const formRef = useTemplateRef('formRef');
const submitted = shallowRef('');
const model = ref({
    contacts: [
        createContact('张三', '13800138000'),
    ],
});
function createContact(name = '', phone = '') {
    return { name, phone };
}
function addContact() {
    model.value.contacts.push(createContact());
    formRef.value?.clearValidate();
    submitted.value = '';
}
function removeContact(index) {
    if (model.value.contacts.length <= 1) {
        return;
    }
    model.value.contacts.splice(index, 1);
    formRef.value?.clearValidate();
    submitted.value = '';
}
function validateName(value) {
    return value?.trim() ? true : '请输入联系人姓名';
}
function validatePhone(value) {
    return /^1[3-9]\d{9}$/.test(value ?? '') ? true : '请输入有效手机号';
}
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
    ...{ class: "gap-4 grid max-w-170" },
});
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
/** @type {__VLS_StyleScopedClasses['max-w-170']} */ ;
const __VLS_0 = FaForm || FaForm;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onSubmit': {} },
    ref: "formRef",
    model: (__VLS_ctx.model),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onSubmit': {} },
    ref: "formRef",
    model: (__VLS_ctx.model),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.submit} */
    onSubmit: (__VLS_ctx.handleSubmit),
};
var __VLS_7;
const { default: __VLS_9 } = __VLS_3.slots;
for (const [_contact, index] of __VLS_vFor((__VLS_ctx.model.contacts))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        key: (index),
        ...{ class: "p-3 border rounded-md gap-3 grid" },
    });
    /** @type {__VLS_StyleScopedClasses['p-3']} */ ;
    /** @type {__VLS_StyleScopedClasses['border']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-3']} */ ;
    /** @type {__VLS_StyleScopedClasses['grid']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex gap-3 items-center justify-between" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-3']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-between']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-sm font-medium" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
    (index + 1);
    const __VLS_10 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
        ...{ 'onClick': {} },
        type: "button",
        variant: "outline",
        size: "sm",
        disabled: (__VLS_ctx.model.contacts.length <= 1),
    }));
    const __VLS_12 = __VLS_11({
        ...{ 'onClick': {} },
        type: "button",
        variant: "outline",
        size: "sm",
        disabled: (__VLS_ctx.model.contacts.length <= 1),
    }, ...__VLS_functionalComponentArgsRest(__VLS_11));
    let __VLS_15;
    const __VLS_16 = {
        /** @type {typeof __VLS_15.click} */
        onClick: (...[$event]) => {
            return (__VLS_ctx.removeContact(index));
            // @ts-ignore
            [model, model, model, handleSubmit, removeContact,];
        },
    };
    const { default: __VLS_17 } = __VLS_13.slots;
    // @ts-ignore
    [];
    var __VLS_13;
    var __VLS_14;
    const __VLS_18 = FaFormItem || FaFormItem;
    // @ts-ignore
    const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
        name: (`contacts[${index}].name`),
        label: "姓名",
        required: true,
        rules: (__VLS_ctx.validateName),
    }));
    const __VLS_20 = __VLS_19({
        name: (`contacts[${index}].name`),
        label: "姓名",
        required: true,
        rules: (__VLS_ctx.validateName),
    }, ...__VLS_functionalComponentArgsRest(__VLS_19));
    const { default: __VLS_23 } = __VLS_21.slots;
    const __VLS_24 = FaInput;
    // @ts-ignore
    const __VLS_25 = __VLS_asFunctionalComponent1(__VLS_24, new __VLS_24({
        placeholder: "请输入姓名",
        ...{ class: "w-full" },
    }));
    const __VLS_26 = __VLS_25({
        placeholder: "请输入姓名",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_25));
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    // @ts-ignore
    [validateName,];
    var __VLS_21;
    const __VLS_29 = FaFormItem || FaFormItem;
    // @ts-ignore
    const __VLS_30 = __VLS_asFunctionalComponent1(__VLS_29, new __VLS_29({
        name: (`contacts[${index}].phone`),
        label: "手机号",
        required: true,
        rules: (__VLS_ctx.validatePhone),
    }));
    const __VLS_31 = __VLS_30({
        name: (`contacts[${index}].phone`),
        label: "手机号",
        required: true,
        rules: (__VLS_ctx.validatePhone),
    }, ...__VLS_functionalComponentArgsRest(__VLS_30));
    const { default: __VLS_34 } = __VLS_32.slots;
    const __VLS_35 = FaInput;
    // @ts-ignore
    const __VLS_36 = __VLS_asFunctionalComponent1(__VLS_35, new __VLS_35({
        type: "tel",
        placeholder: "请输入手机号",
        ...{ class: "w-full" },
    }));
    const __VLS_37 = __VLS_36({
        type: "tel",
        placeholder: "请输入手机号",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_36));
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    // @ts-ignore
    [validatePhone,];
    var __VLS_32;
    // @ts-ignore
    [];
}
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-wrap gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_40 = FaButton || FaButton;
// @ts-ignore
const __VLS_41 = __VLS_asFunctionalComponent1(__VLS_40, new __VLS_40({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}));
const __VLS_42 = __VLS_41({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_41));
let __VLS_45;
const __VLS_46 = {
    /** @type {typeof __VLS_45.click} */
    onClick: (__VLS_ctx.addContact),
};
const { default: __VLS_47 } = __VLS_43.slots;
// @ts-ignore
[addContact,];
var __VLS_43;
var __VLS_44;
const __VLS_48 = FaButton || FaButton;
// @ts-ignore
const __VLS_49 = __VLS_asFunctionalComponent1(__VLS_48, new __VLS_48({
    type: "submit",
}));
const __VLS_50 = __VLS_49({
    type: "submit",
}, ...__VLS_functionalComponentArgsRest(__VLS_49));
const { default: __VLS_53 } = __VLS_51.slots;
// @ts-ignore
[];
var __VLS_51;
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
var __VLS_8 = __VLS_7;
// @ts-ignore
[submitted, submitted,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
