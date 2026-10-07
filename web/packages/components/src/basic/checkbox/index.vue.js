import { computed, useId, watch } from 'vue';
import { cn } from '#utils';
import { Label } from '../label/label';
import { Checkbox } from './checkbox';
defineOptions({
    name: 'BuiltInCheckbox',
});
const props = defineProps();
const emit = defineEmits();
const value = defineModel();
const generatedId = useId();
const checkboxId = computed(() => props.id || generatedId);
watch(value, (newValue) => {
    emit('change', newValue);
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
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: (__VLS_ctx.cn('flex-center-start gap-2', props.class)) },
});
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Checkbox} */
Checkbox;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    id: (__VLS_ctx.checkboxId),
    modelValue: (__VLS_ctx.value),
    disabled: (__VLS_ctx.disabled),
    ...{ class: (props.itemClass) },
}));
const __VLS_2 = __VLS_1({
    id: (__VLS_ctx.checkboxId),
    modelValue: (__VLS_ctx.value),
    disabled: (__VLS_ctx.disabled),
    ...{ class: (props.itemClass) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
/** @ts-ignore @type { | typeof __VLS_components.Label | typeof __VLS_components.Label} */
Label;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    for: (__VLS_ctx.checkboxId),
    ...{ class: (__VLS_ctx.cn('text-sm cursor-pointer empty:hidden', props.disabled && 'cursor-not-allowed opacity-60', props.labelClass)) },
}));
const __VLS_7 = __VLS_6({
    for: (__VLS_ctx.checkboxId),
    ...{ class: (__VLS_ctx.cn('text-sm cursor-pointer empty:hidden', props.disabled && 'cursor-not-allowed opacity-60', props.labelClass)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
const { default: __VLS_10 } = __VLS_8.slots;
var __VLS_11 = {};
// @ts-ignore
[cn, cn, checkboxId, checkboxId, value, disabled,];
var __VLS_8;
// @ts-ignore
var __VLS_12 = __VLS_11;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
