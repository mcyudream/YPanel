import { useSlots } from 'vue';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from './collapsible';
defineOptions({
    name: 'BuiltInCollapsible',
});
const open = defineModel('modelValue', {
    default: false,
});
const slots = useSlots();
const __VLS_defaultModels = {
    'modelValue': false,
};
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Collapsible | typeof __VLS_components.Collapsible} */
Collapsible;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    open: (__VLS_ctx.open),
}));
const __VLS_2 = __VLS_1({
    open: (__VLS_ctx.open),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
if (!!__VLS_ctx.slots.trigger) {
    let __VLS_7;
    /** @ts-ignore @type { | typeof __VLS_components.CollapsibleTrigger | typeof __VLS_components.CollapsibleTrigger} */
    CollapsibleTrigger;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({}));
    const __VLS_9 = __VLS_8({}, ...__VLS_functionalComponentArgsRest(__VLS_8));
    const { default: __VLS_12 } = __VLS_10.slots;
    var __VLS_13 = {
        open: __VLS_ctx.open,
    };
    // @ts-ignore
    [open, open, slots,];
    var __VLS_10;
}
let __VLS_15;
/** @ts-ignore @type { | typeof __VLS_components.CollapsibleContent | typeof __VLS_components.CollapsibleContent} */
CollapsibleContent;
// @ts-ignore
const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({}));
const __VLS_17 = __VLS_16({}, ...__VLS_functionalComponentArgsRest(__VLS_16));
const { default: __VLS_20 } = __VLS_18.slots;
var __VLS_21 = {};
// @ts-ignore
[];
var __VLS_18;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_14 = __VLS_13, __VLS_22 = __VLS_21;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
