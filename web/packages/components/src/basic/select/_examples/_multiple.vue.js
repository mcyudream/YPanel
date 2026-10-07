import { ref } from 'vue';
import FaSelect from '../index.vue';
const select = ref([]);
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
const __VLS_0 = FaSelect;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.select),
    options: (__VLS_ctx.options),
    multiple: true,
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.select),
    options: (__VLS_ctx.options),
    multiple: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[select, options,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
