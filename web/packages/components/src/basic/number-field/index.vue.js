import { cn } from '#utils';
import { NumberField, NumberFieldContent, NumberFieldDecrement, NumberFieldIncrement, NumberFieldInput, } from './number-field';
const props = defineProps();
const modelValue = defineModel({
    required: true,
});
let __VLS_modelEmit;
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
/** @ts-ignore @type { | typeof __VLS_components.NumberField | typeof __VLS_components.NumberField} */
NumberField;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.modelValue),
    min: (props.min),
    max: (props.max),
    step: (props.step),
    disabled: (props.disabled),
    ...{ class: (__VLS_ctx.cn('w-[200px]', props.class)) },
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.modelValue),
    min: (props.min),
    max: (props.max),
    step: (props.step),
    disabled: (props.disabled),
    ...{ class: (__VLS_ctx.cn('w-[200px]', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.NumberFieldContent | typeof __VLS_components.NumberFieldContent} */
NumberFieldContent;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({}));
const __VLS_9 = __VLS_8({}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
let __VLS_13;
/** @ts-ignore @type { | typeof __VLS_components.NumberFieldDecrement} */
NumberFieldDecrement;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({}));
const __VLS_15 = __VLS_14({}, ...__VLS_functionalComponentArgsRest(__VLS_14));
let __VLS_18;
/** @ts-ignore @type { | typeof __VLS_components.NumberFieldInput} */
NumberFieldInput;
// @ts-ignore
const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({}));
const __VLS_20 = __VLS_19({}, ...__VLS_functionalComponentArgsRest(__VLS_19));
let __VLS_23;
/** @ts-ignore @type { | typeof __VLS_components.NumberFieldIncrement} */
NumberFieldIncrement;
// @ts-ignore
const __VLS_24 = __VLS_asFunctionalComponent1(__VLS_23, new __VLS_23({}));
const __VLS_25 = __VLS_24({}, ...__VLS_functionalComponentArgsRest(__VLS_24));
// @ts-ignore
[modelValue, cn,];
var __VLS_10;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
export default {};
