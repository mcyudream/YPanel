import { cn } from '#utils';
import { Label } from '../../label/label';
import { useFormField } from './useFormField';
const props = defineProps();
const { error, formItemId } = useFormField();
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Label | typeof __VLS_components.Label} */
Label;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "form-label",
    dataError: (!!__VLS_ctx.error),
    ...{ class: (__VLS_ctx.cn('data-[error=true]:text-destructive', props.class)) },
    for: (__VLS_ctx.formItemId),
}));
const __VLS_2 = __VLS_1({
    dataSlot: "form-label",
    dataError: (!!__VLS_ctx.error),
    ...{ class: (__VLS_ctx.cn('data-[error=true]:text-destructive', props.class)) },
    for: (__VLS_ctx.formItemId),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
var __VLS_7 = {};
// @ts-ignore
[error, cn, formItemId,];
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
