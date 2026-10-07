import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './tooltip';
defineOptions({
    name: 'BuiltInTooltip',
});
const __VLS_props = withDefaults(defineProps(), {
    text: '',
    delay: 300,
});
const __VLS_defaults = {
    text: '',
    delay: 300,
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.TooltipProvider | typeof __VLS_components.TooltipProvider} */
TooltipProvider;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    delayDuration: (__VLS_ctx.delay),
    disabled: __VLS_ctx.disabled,
}));
const __VLS_2 = __VLS_1({
    delayDuration: (__VLS_ctx.delay),
    disabled: __VLS_ctx.disabled,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.Tooltip | typeof __VLS_components.Tooltip} */
Tooltip;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({}));
const __VLS_9 = __VLS_8({}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
let __VLS_13;
/** @ts-ignore @type { | typeof __VLS_components.TooltipTrigger | typeof __VLS_components.TooltipTrigger} */
TooltipTrigger;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
    asChild: true,
}));
const __VLS_15 = __VLS_14({
    asChild: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_14));
const { default: __VLS_18 } = __VLS_16.slots;
var __VLS_19 = {};
// @ts-ignore
[delay, disabled,];
var __VLS_16;
let __VLS_21;
/** @ts-ignore @type { | typeof __VLS_components.TooltipContent | typeof __VLS_components.TooltipContent} */
TooltipContent;
// @ts-ignore
const __VLS_22 = __VLS_asFunctionalComponent1(__VLS_21, new __VLS_21({
    side: __VLS_ctx.side,
    align: __VLS_ctx.align,
    ...{ class: "text-xs z-10000" },
}));
const __VLS_23 = __VLS_22({
    side: __VLS_ctx.side,
    align: __VLS_ctx.align,
    ...{ class: "text-xs z-10000" },
}, ...__VLS_functionalComponentArgsRest(__VLS_22));
/** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
/** @type {__VLS_StyleScopedClasses['z-10000']} */ ;
const { default: __VLS_26 } = __VLS_24.slots;
var __VLS_27 = {};
__VLS_asFunctionalElement1(__VLS_intrinsics.p, __VLS_intrinsics.p)({});
(__VLS_ctx.text);
// @ts-ignore
[side, align, text,];
var __VLS_24;
// @ts-ignore
[];
var __VLS_10;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_20 = __VLS_19, __VLS_28 = __VLS_27;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
