import Icon from '../icon/index.vue';
import { Button } from './button';
defineOptions({
    name: 'BuiltInButton',
});
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Button | typeof __VLS_components.Button} */
Button;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    variant: __VLS_ctx.variant,
    size: __VLS_ctx.size,
    disabled: (props.disabled || props.loading),
    ...{ class: (props.class) },
}));
const __VLS_2 = __VLS_1({
    variant: __VLS_ctx.variant,
    size: __VLS_ctx.size,
    disabled: (props.disabled || props.loading),
    ...{ class: (props.class) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
if (__VLS_ctx.loading) {
    const __VLS_7 = Icon;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
        name: "i-line-md:loading-twotone-loop",
    }));
    const __VLS_9 = __VLS_8({
        name: "i-line-md:loading-twotone-loop",
    }, ...__VLS_functionalComponentArgsRest(__VLS_8));
}
var __VLS_12 = {};
// @ts-ignore
[variant, size, loading,];
var __VLS_3;
// @ts-ignore
var __VLS_13 = __VLS_12;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
