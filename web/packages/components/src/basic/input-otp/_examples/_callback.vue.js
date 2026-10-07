import { ref } from 'vue';
import { useToast } from '../../toast';
import FaInputOTP from '../index.vue';
const input = ref('');
const inputText = ref('等待输入');
const completeText = ref('等待完成');
const toast = useToast();
function handleInput(value) {
    inputText.value = `input: ${value || '空值'}`;
}
function handleComplete(value) {
    completeText.value = `complete: ${value}`;
    toast(value);
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "space-y-3" },
});
/** @type {__VLS_StyleScopedClasses['space-y-3']} */ ;
const __VLS_0 = FaInputOTP;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onInput': {} },
    ...{ 'onComplete': {} },
    modelValue: (__VLS_ctx.input),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onInput': {} },
    ...{ 'onComplete': {} },
    modelValue: (__VLS_ctx.input),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.input} */
    onInput: (__VLS_ctx.handleInput),
};
const __VLS_7 = {
    /** @type {typeof __VLS_5.complete} */
    onComplete: (__VLS_ctx.handleComplete),
};
var __VLS_3;
var __VLS_4;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground space-y-1" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
/** @type {__VLS_StyleScopedClasses['space-y-1']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
(__VLS_ctx.inputText);
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
(__VLS_ctx.completeText);
// @ts-ignore
[input, handleInput, handleComplete, inputText, completeText,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
