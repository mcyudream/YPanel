import { ref } from 'vue';
import FaSelect from '../index.vue';
const popperValue = ref('2');
const itemAlignedValue = ref('2');
const options = [
    { label: 'Option 1', value: '1' },
    { label: 'Option 2', value: '2' },
    { label: 'Option 3', value: '3' },
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "space-y-2" },
});
/** @type {__VLS_StyleScopedClasses['space-y-2']} */ ;
const __VLS_0 = FaSelect;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.popperValue),
    options: (__VLS_ctx.options),
    position: "popper",
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.popperValue),
    options: (__VLS_ctx.options),
    position: "popper",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = FaSelect;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    modelValue: (__VLS_ctx.itemAlignedValue),
    options: (__VLS_ctx.options),
    position: "item-aligned",
}));
const __VLS_7 = __VLS_6({
    modelValue: (__VLS_ctx.itemAlignedValue),
    options: (__VLS_ctx.options),
    position: "item-aligned",
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
// @ts-ignore
[popperValue, options, options, itemAlignedValue,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
