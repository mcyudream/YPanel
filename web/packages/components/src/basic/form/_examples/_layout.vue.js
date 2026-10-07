import { ref, shallowRef } from 'vue';
import FaInput from '../../input/index.vue';
import FaNumberField from '../../number-field/index.vue';
import FaRadioGroup from '../../radio-group/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const labelPlacement = shallowRef('left');
const labelWidth = shallowRef(150);
const model = ref({
    name: 'Fantastic-admin',
    email: 'admin@example.com',
    phone: '13800138000',
});
const labelPlacementOptions = [
    { label: 'top', value: 'top' },
    { label: 'left', value: 'left' },
    { label: 'right', value: 'right' },
];
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
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-4 grid sm:grid-cols-[minmax(0,1fr)_200px]" },
});
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
/** @type {__VLS_StyleScopedClasses['sm:grid-cols-[minmax(0,1fr)_200px]']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-2 grid" },
});
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm font-medium" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
const __VLS_0 = FaRadioGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.labelPlacement),
    options: (__VLS_ctx.labelPlacementOptions),
    ...{ class: "flex flex-wrap gap-4" },
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.labelPlacement),
    options: (__VLS_ctx.labelPlacementOptions),
    ...{ class: "flex flex-wrap gap-4" },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-2 grid" },
});
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm font-medium" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
const __VLS_5 = FaNumberField;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    modelValue: (__VLS_ctx.labelWidth),
    min: (80),
    max: (240),
    step: (10),
    ...{ class: "w-full" },
}));
const __VLS_7 = __VLS_6({
    modelValue: (__VLS_ctx.labelWidth),
    min: (80),
    max: (240),
    step: (10),
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
const __VLS_10 = FaForm || FaForm;
// @ts-ignore
const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
    model: (__VLS_ctx.model),
    labelPlacement: (__VLS_ctx.labelPlacement),
    labelWidth: (__VLS_ctx.labelWidth),
}));
const __VLS_12 = __VLS_11({
    model: (__VLS_ctx.model),
    labelPlacement: (__VLS_ctx.labelPlacement),
    labelWidth: (__VLS_ctx.labelWidth),
}, ...__VLS_functionalComponentArgsRest(__VLS_11));
const { default: __VLS_15 } = __VLS_13.slots;
const __VLS_16 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
    name: "name",
    label: "项目名称",
}));
const __VLS_18 = __VLS_17({
    name: "name",
    label: "项目名称",
}, ...__VLS_functionalComponentArgsRest(__VLS_17));
const { default: __VLS_21 } = __VLS_19.slots;
const __VLS_22 = FaInput;
// @ts-ignore
const __VLS_23 = __VLS_asFunctionalComponent1(__VLS_22, new __VLS_22({
    ...{ class: "w-full" },
}));
const __VLS_24 = __VLS_23({
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_23));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[labelPlacement, labelPlacement, labelPlacementOptions, labelWidth, labelWidth, model,];
var __VLS_19;
const __VLS_27 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_28 = __VLS_asFunctionalComponent1(__VLS_27, new __VLS_27({
    name: "email",
    label: "邮箱地址",
}));
const __VLS_29 = __VLS_28({
    name: "email",
    label: "邮箱地址",
}, ...__VLS_functionalComponentArgsRest(__VLS_28));
const { default: __VLS_32 } = __VLS_30.slots;
const __VLS_33 = FaInput;
// @ts-ignore
const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({
    type: "email",
    ...{ class: "w-full" },
}));
const __VLS_35 = __VLS_34({
    type: "email",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_34));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[];
var __VLS_30;
const __VLS_38 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({
    name: "phone",
    label: "联系电话",
}));
const __VLS_40 = __VLS_39({
    name: "phone",
    label: "联系电话",
}, ...__VLS_functionalComponentArgsRest(__VLS_39));
const { default: __VLS_43 } = __VLS_41.slots;
const __VLS_44 = FaInput;
// @ts-ignore
const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
    type: "tel",
    ...{ class: "w-full" },
}));
const __VLS_46 = __VLS_45({
    type: "tel",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_45));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[];
var __VLS_41;
// @ts-ignore
[];
var __VLS_13;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
