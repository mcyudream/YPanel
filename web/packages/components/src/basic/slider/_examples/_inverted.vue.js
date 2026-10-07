import { ref } from 'vue';
import FaSlider from '../index.vue';
const normalValue = ref([30]);
const invertedValue = ref([30]);
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "space-y-6" },
});
/** @type {__VLS_StyleScopedClasses['space-y-6']} */ ;
const __VLS_0 = FaSlider;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.normalValue),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.normalValue),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = FaSlider;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    modelValue: (__VLS_ctx.invertedValue),
    inverted: true,
}));
const __VLS_7 = __VLS_6({
    modelValue: (__VLS_ctx.invertedValue),
    inverted: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
// @ts-ignore
[normalValue, invertedValue,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
