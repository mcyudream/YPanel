import { ErrorMessage } from 'vee-validate';
import { toValue } from 'vue';
import { cn } from '#utils';
import { useFormField } from './useFormField';
const props = defineProps();
const __VLS_slots = defineSlots();
const { name, formMessageId } = useFormField();
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
/** @ts-ignore @type { | typeof __VLS_components.ErrorMessage | typeof __VLS_components.ErrorMessage} */
ErrorMessage;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    id: (__VLS_ctx.formMessageId),
    dataSlot: "form-message",
    as: "p",
    name: (__VLS_ctx.toValue(__VLS_ctx.name)),
    ...{ class: (__VLS_ctx.cn('text-destructive text-sm', props.class)) },
}));
const __VLS_2 = __VLS_1({
    id: (__VLS_ctx.formMessageId),
    dataSlot: "form-message",
    as: "p",
    name: (__VLS_ctx.toValue(__VLS_ctx.name)),
    ...{ class: (__VLS_ctx.cn('text-destructive text-sm', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { default: __VLS_7 } = __VLS_3.slots;
    const [{ message }] = __VLS_vSlot(__VLS_7);
    __VLS_asFunctionalSlot(__VLS_slots['default'])({
        message: (message),
    });
    (message);
    // @ts-ignore
    [formMessageId, toValue, name, cn,];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
