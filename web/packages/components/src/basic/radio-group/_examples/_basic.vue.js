import { ref } from 'vue';
import FaRadioGroup from '../index.vue';
const value = ref('comfortable');
const options = [
    { label: '默认', value: 'default' },
    { label: '舒适', value: 'comfortable' },
    { label: '紧凑', value: 'compact' },
];
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
const __VLS_0 = FaRadioGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = FaRadioGroup;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
    ...{ class: "flex" },
}));
const __VLS_7 = __VLS_6({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
    ...{ class: "flex" },
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
(__VLS_ctx.value);
// @ts-ignore
[value, value, value, options, options,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
