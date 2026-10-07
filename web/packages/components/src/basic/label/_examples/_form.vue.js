import { reactive } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaSelect from '../../select/index.vue';
import FaTextarea from '../../textarea/index.vue';
import FaLabel from '../index.vue';
const form = reactive({
    title: '',
    type: 'notice',
    description: '',
});
const typeOptions = [
    { label: '公告', value: 'notice' },
    { label: '消息', value: 'message' },
    { label: '任务', value: 'task' },
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "max-w-xl space-y-4" },
});
/** @type {__VLS_StyleScopedClasses['max-w-xl']} */ ;
/** @type {__VLS_StyleScopedClasses['space-y-4']} */ ;
const __VLS_0 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    label: "标题",
    labelWidth: "5rem",
}));
const __VLS_2 = __VLS_1({
    label: "标题",
    labelWidth: "5rem",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
const __VLS_6 = FaInput;
// @ts-ignore
const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({
    modelValue: (__VLS_ctx.form.title),
    placeholder: "请输入标题",
    ...{ class: "w-full" },
}));
const __VLS_8 = __VLS_7({
    modelValue: (__VLS_ctx.form.title),
    placeholder: "请输入标题",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_7));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[form,];
var __VLS_3;
const __VLS_11 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_12 = __VLS_asFunctionalComponent1(__VLS_11, new __VLS_11({
    label: "类型",
    labelWidth: "5rem",
}));
const __VLS_13 = __VLS_12({
    label: "类型",
    labelWidth: "5rem",
}, ...__VLS_functionalComponentArgsRest(__VLS_12));
const { default: __VLS_16 } = __VLS_14.slots;
const __VLS_17 = FaSelect;
// @ts-ignore
const __VLS_18 = __VLS_asFunctionalComponent1(__VLS_17, new __VLS_17({
    modelValue: (__VLS_ctx.form.type),
    options: (__VLS_ctx.typeOptions),
    ...{ class: "w-full" },
}));
const __VLS_19 = __VLS_18({
    modelValue: (__VLS_ctx.form.type),
    options: (__VLS_ctx.typeOptions),
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_18));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[form, typeOptions,];
var __VLS_14;
const __VLS_22 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_23 = __VLS_asFunctionalComponent1(__VLS_22, new __VLS_22({
    label: "描述",
    labelWidth: "5rem",
    ...{ class: "items-start" },
}));
const __VLS_24 = __VLS_23({
    label: "描述",
    labelWidth: "5rem",
    ...{ class: "items-start" },
}, ...__VLS_functionalComponentArgsRest(__VLS_23));
/** @type {__VLS_StyleScopedClasses['items-start']} */ ;
const { default: __VLS_27 } = __VLS_25.slots;
const __VLS_28 = FaTextarea;
// @ts-ignore
const __VLS_29 = __VLS_asFunctionalComponent1(__VLS_28, new __VLS_28({
    modelValue: (__VLS_ctx.form.description),
    placeholder: "请输入描述",
    ...{ class: "w-full" },
}));
const __VLS_30 = __VLS_29({
    modelValue: (__VLS_ctx.form.description),
    placeholder: "请输入描述",
    ...{ class: "w-full" },
}, ...__VLS_functionalComponentArgsRest(__VLS_29));
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
// @ts-ignore
[form,];
var __VLS_25;
const __VLS_33 = FaLabel || FaLabel;
// @ts-ignore
const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({
    labelWidth: "5rem",
}));
const __VLS_35 = __VLS_34({
    labelWidth: "5rem",
}, ...__VLS_functionalComponentArgsRest(__VLS_34));
const { default: __VLS_38 } = __VLS_36.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_39 = FaButton || FaButton;
// @ts-ignore
const __VLS_40 = __VLS_asFunctionalComponent1(__VLS_39, new __VLS_39({}));
const __VLS_41 = __VLS_40({}, ...__VLS_functionalComponentArgsRest(__VLS_40));
const { default: __VLS_44 } = __VLS_42.slots;
// @ts-ignore
[];
var __VLS_42;
const __VLS_45 = FaButton || FaButton;
// @ts-ignore
const __VLS_46 = __VLS_asFunctionalComponent1(__VLS_45, new __VLS_45({
    variant: "outline",
}));
const __VLS_47 = __VLS_46({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_46));
const { default: __VLS_50 } = __VLS_48.slots;
// @ts-ignore
[];
var __VLS_48;
// @ts-ignore
[];
var __VLS_36;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
