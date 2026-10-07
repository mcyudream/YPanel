import { useId } from 'reka-ui';
import { provide } from 'vue';
import { cn } from '#utils';
import { FORM_ITEM_INJECTION_KEY } from './injectionKeys';
const props = defineProps();
const id = useId();
provide(FORM_ITEM_INJECTION_KEY, id);
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    'data-slot': "form-item",
    ...{ class: (__VLS_ctx.cn('grid gap-2', props.class)) },
});
var __VLS_0 = {};
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[cn,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
