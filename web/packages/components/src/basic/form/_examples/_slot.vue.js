import { ref, shallowRef } from 'vue';
import FaButton from '../../button/index.vue';
import { Checkbox as FaCheckbox } from '../../checkbox/checkbox';
import FaInput from '../../input/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const result = shallowRef('');
const model = ref({
    email: '',
    agreement: false,
});
const validationSchema = {
    email(value) {
        return /^\S[^\s@]*@\S[^\s.]*\.\S+$/.test(value) ? true : '请输入有效邮箱';
    },
    agreement(value) {
        return value ? true : '请先同意服务协议';
    },
};
function handleSubmit(values) {
    result.value = JSON.stringify(values, null, 2);
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
    name: "email",
    required: true,
}));
const __VLS_10 = __VLS_9({
    name: "email",
    required: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
const { default: __VLS_13 } = __VLS_11.slots;
{
    const { label: __VLS_14 } = __VLS_11.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "text-destructive" },
    });
    /** @type {__VLS_StyleScopedClasses['text-destructive']} */ ;
    // @ts-ignore
    [model, validationSchema, handleSubmit,];
}
const __VLS_15 = FaInput;
// @ts-ignore
const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({
    type: "email",
    placeholder: "name@example.com",
    ...{ class: "w-full" },
}));
const __VLS_17 = __VLS_16({
    type: "email",
    placeholder: "name@example.com",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_16));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
{
    const { description: __VLS_20 } = __VLS_11.slots;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_11;
const __VLS_21 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_22 = __VLS_asFunctionalComponent1(__VLS_21, new __VLS_21({
    name: "agreement",
}));
const __VLS_23 = __VLS_22({
    name: "agreement",
}, ...__VLS_functionalComponentArgsRest(__VLS_22));
const { default: __VLS_26 } = __VLS_24.slots;
{
    const { default: __VLS_27 } = __VLS_24.slots;
    const [{ componentField }] = __VLS_vSlot(__VLS_27);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex gap-2 items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    let __VLS_28;
    /** @ts-ignore @type { | typeof __VLS_components.FaCheckbox} */
    FaCheckbox;
    // @ts-ignore
    const __VLS_29 = __VLS_asFunctionalComponent1(__VLS_28, new __VLS_28({
        ...{ 'onUpdate:modelValue': {} },
        modelValue: componentField.modelValue,
    }));
    const __VLS_30 = __VLS_29({
        ...{ 'onUpdate:modelValue': {} },
        modelValue: componentField.modelValue,
    }, ...__VLS_functionalComponentArgsRest(__VLS_29));
    let __VLS_33;
    const __VLS_34 = {
        /** @type {typeof __VLS_33.'update:modelValue'} */
        'onUpdate:modelValue': (...[$event]) => {
            return (componentField['onUpdate:modelValue']?.($event));
            // @ts-ignore
            [];
        },
    };
    var __VLS_31;
    var __VLS_32;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "text-sm" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_24;
const __VLS_35 = FaButton || FaButton;
// @ts-ignore
const __VLS_36 = __VLS_asFunctionalComponent1(__VLS_35, new __VLS_35({
    type: "submit",
}));
const __VLS_37 = __VLS_36({
    type: "submit",
}, ...__VLS_functionalComponentArgsRest(__VLS_36));
const { default: __VLS_40 } = __VLS_38.slots;
// @ts-ignore
[];
var __VLS_38;
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
if (__VLS_ctx.result) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.pre, __VLS_intrinsics.pre)({
        ...{ class: "text-sm m-0 p-4 rounded-md bg-muted" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['m-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['p-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-muted']} */ ;
    (__VLS_ctx.result);
}
// @ts-ignore
[result, result,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
