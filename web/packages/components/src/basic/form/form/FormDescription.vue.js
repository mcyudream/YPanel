import { cn } from '#utils';
import { useFormField } from './useFormField';
const props = defineProps();
const { formDescriptionId } = useFormField();
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({
    id: (__VLS_ctx.formDescriptionId),
    'data-slot': "form-description",
    ...{ class: (__VLS_ctx.cn('text-muted-foreground text-sm', props.class)) },
});
var __VLS_0 = {};
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[formDescriptionId, cn,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
