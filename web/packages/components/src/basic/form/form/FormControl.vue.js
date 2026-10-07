import { Slot } from 'reka-ui';
import { useFormField } from './useFormField';
const { error, formItemId, formDescriptionId, formMessageId } = useFormField();
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Slot | typeof __VLS_components.Slot} */
Slot;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    id: (__VLS_ctx.formItemId),
    dataSlot: "form-control",
    'aria-describedby': (!__VLS_ctx.error ? `${__VLS_ctx.formDescriptionId}` : `${__VLS_ctx.formDescriptionId} ${__VLS_ctx.formMessageId}`),
    'aria-invalid': (!!__VLS_ctx.error),
}));
const __VLS_2 = __VLS_1({
    id: (__VLS_ctx.formItemId),
    dataSlot: "form-control",
    'aria-describedby': (!__VLS_ctx.error ? `${__VLS_ctx.formDescriptionId}` : `${__VLS_ctx.formDescriptionId} ${__VLS_ctx.formMessageId}`),
    'aria-invalid': (!!__VLS_ctx.error),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
var __VLS_7 = {};
// @ts-ignore
[formItemId, error, error, formDescriptionId, formDescriptionId, formMessageId,];
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({});
const __VLS_export = {};
export default {};
