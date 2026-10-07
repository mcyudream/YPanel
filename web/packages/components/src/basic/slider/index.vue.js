import { Slider } from './slider';
defineOptions({
    name: 'BuiltInSlider',
});
const props = withDefaults(defineProps(), {
    defaultValue: () => [0],
    disabled: false,
    inverted: false,
    max: 100,
    min: 0,
    step: 1,
    orientation: 'horizontal',
    thumbAlignment: 'contain',
    tooltip: true,
});
const modelValue = defineModel();
let __VLS_modelEmit;
const __VLS_defaults = {
    defaultValue: () => [0],
    disabled: false,
    inverted: false,
    max: 100,
    min: 0,
    step: 1,
    orientation: 'horizontal',
    thumbAlignment: 'contain',
    tooltip: true,
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Slider} */
Slider;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...(props),
    modelValue: (__VLS_ctx.modelValue),
}));
const __VLS_2 = __VLS_1({
    ...(props),
    modelValue: (__VLS_ctx.modelValue),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[modelValue,];
const __VLS_export = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
export default {};
