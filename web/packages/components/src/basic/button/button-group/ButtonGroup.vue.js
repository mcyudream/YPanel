import { cn } from '#utils';
import { buttonGroupVariants } from '.';
const props = defineProps();
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
    role: "group",
    'data-slot': "button-group",
    'data-orientation': (props.orientation),
    ...{ class: (__VLS_ctx.cn(__VLS_ctx.buttonGroupVariants({ orientation: props.orientation }), props.class)) },
});
var __VLS_0 = {};
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[cn, buttonGroupVariants,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
