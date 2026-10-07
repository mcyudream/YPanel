import { cn } from '#utils';
import { inputGroupAddonVariants } from '.';
const props = withDefaults(defineProps(), {
    align: 'inline-start',
});
function handleInputGroupAddonClick(e) {
    const currentTarget = e.currentTarget;
    const target = e.target;
    if (target && target.closest('button')) {
        return;
    }
    if (currentTarget && currentTarget?.parentElement) {
        currentTarget.parentElement?.querySelector('input')?.focus();
    }
}
const __VLS_defaults = {
    align: 'inline-start',
};
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
    ...{ onClick: (__VLS_ctx.handleInputGroupAddonClick) },
    role: "group",
    'data-slot': "input-group-addon",
    'data-align': (props.align),
    ...{ class: (__VLS_ctx.cn(__VLS_ctx.inputGroupAddonVariants({ align: props.align }), props.class)) },
});
var __VLS_0 = {};
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[handleInputGroupAddonClick, cn, inputGroupAddonVariants,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
