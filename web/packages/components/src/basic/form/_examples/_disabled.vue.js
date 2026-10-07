import { ref, shallowRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaSelect from '../../select/index.vue';
import FaSwitch from '../../switch/index.vue';
import FaTextarea from '../../textarea/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const disabled = shallowRef(true);
const submitted = shallowRef('');
const model = ref({
    name: 'Fantastic-admin',
    role: 'admin',
    remark: '禁用状态下，表单项内的控件会同步不可编辑。',
});
const roleOptions = [
    { label: '管理员', value: 'admin' },
    { label: '运营', value: 'operator' },
    { label: '访客', value: 'guest' },
];
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
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex gap-2 items-center" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
const __VLS_0 = FaSwitch;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.disabled),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.disabled),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
    ...{ class: "text-sm" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
const __VLS_5 = FaForm || FaForm;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    ...{ 'onSubmit': {} },
    disabled: (__VLS_ctx.disabled),
    model: (__VLS_ctx.model),
}));
const __VLS_7 = __VLS_6({
    ...{ 'onSubmit': {} },
    disabled: (__VLS_ctx.disabled),
    model: (__VLS_ctx.model),
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
let __VLS_10;
const __VLS_11 = {
    /** @type {typeof __VLS_10.submit} */
    onSubmit: (__VLS_ctx.handleSubmit),
};
const { default: __VLS_12 } = __VLS_8.slots;
const __VLS_13 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
    name: "name",
    label: "项目名称",
}));
const __VLS_15 = __VLS_14({
    name: "name",
    label: "项目名称",
}, ...__VLS_functionalComponentArgsRest(__VLS_14));
const { default: __VLS_18 } = __VLS_16.slots;
const __VLS_19 = FaInput;
// @ts-ignore
const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
    ...{ class: "w-full" },
}));
const __VLS_21 = __VLS_20({
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_20));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[disabled, disabled, model, handleSubmit,];
var __VLS_16;
const __VLS_24 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_25 = __VLS_asFunctionalComponent1(__VLS_24, new __VLS_24({
    name: "role",
    label: "角色",
}));
const __VLS_26 = __VLS_25({
    name: "role",
    label: "角色",
}, ...__VLS_functionalComponentArgsRest(__VLS_25));
const { default: __VLS_29 } = __VLS_27.slots;
const __VLS_30 = FaSelect;
// @ts-ignore
const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({
    options: (__VLS_ctx.roleOptions),
    ...{ class: "w-full" },
}));
const __VLS_32 = __VLS_31({
    options: (__VLS_ctx.roleOptions),
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_31));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[roleOptions,];
var __VLS_27;
const __VLS_35 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_36 = __VLS_asFunctionalComponent1(__VLS_35, new __VLS_35({
    name: "remark",
    label: "备注",
}));
const __VLS_37 = __VLS_36({
    name: "remark",
    label: "备注",
}, ...__VLS_functionalComponentArgsRest(__VLS_36));
const { default: __VLS_40 } = __VLS_38.slots;
const __VLS_41 = FaTextarea;
// @ts-ignore
const __VLS_42 = __VLS_asFunctionalComponent1(__VLS_41, new __VLS_41({
    ...{ class: "w-full" },
}));
const __VLS_43 = __VLS_42({
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_42));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[];
var __VLS_38;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_46 = FaButton || FaButton;
// @ts-ignore
const __VLS_47 = __VLS_asFunctionalComponent1(__VLS_46, new __VLS_46({
    type: "submit",
    disabled: (__VLS_ctx.disabled),
}));
const __VLS_48 = __VLS_47({
    type: "submit",
    disabled: (__VLS_ctx.disabled),
}, ...__VLS_functionalComponentArgsRest(__VLS_47));
const { default: __VLS_51 } = __VLS_49.slots;
// @ts-ignore
[disabled,];
var __VLS_49;
const __VLS_52 = FaButton || FaButton;
// @ts-ignore
const __VLS_53 = __VLS_asFunctionalComponent1(__VLS_52, new __VLS_52({
    type: "reset",
    variant: "outline",
    disabled: (__VLS_ctx.disabled),
}));
const __VLS_54 = __VLS_53({
    type: "reset",
    variant: "outline",
    disabled: (__VLS_ctx.disabled),
}, ...__VLS_functionalComponentArgsRest(__VLS_53));
const { default: __VLS_57 } = __VLS_55.slots;
// @ts-ignore
[disabled,];
var __VLS_55;
// @ts-ignore
[];
var __VLS_8;
var __VLS_9;
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
