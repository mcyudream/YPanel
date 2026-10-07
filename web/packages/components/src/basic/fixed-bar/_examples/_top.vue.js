import { ref } from 'vue';
import FaSlider from '../../slider/index.vue';
import FaFixedBar from '../index.vue';
const height = ref([50]);
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaFixedBar || FaFixedBar;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    position: "top",
}));
const __VLS_2 = __VLS_1({
    position: "top",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
const __VLS_7 = FaSlider;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    modelValue: (__VLS_ctx.height),
}));
const __VLS_9 = __VLS_8({
    modelValue: (__VLS_ctx.height),
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
__VLS_asFunctionalElement1(__VLS_intrinsics.div)({
    ...{ style: (`height: ${__VLS_ctx.height[0]}px;`) },
});
// @ts-ignore
[height, height,];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
