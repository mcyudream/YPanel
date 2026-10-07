import { computed, shallowRef } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaCheckboxGroup from '../CheckboxGroup.vue';
const value = shallowRef(['dashboard']);
const options = [
    { label: '看板订阅', value: 'dashboard' },
    { label: '周报摘要', value: 'report' },
    { label: '系统告警', value: 'alarm' },
];
const currentText = computed(() => options
    .filter(option => value.value.includes(option.value))
    .map(option => option.label)
    .join('、'));
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-4 grid" },
});
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
const __VLS_0 = FaCheckboxGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
    min: (1),
    max: (2),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
    min: (1),
    max: (2),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
(__VLS_ctx.currentText || '未选择');
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-xs text-muted-foreground leading-5" },
});
/** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
/** @type {__VLS_StyleScopedClasses['leading-5']} */ ;
// @ts-ignore
[value, options, currentText,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
