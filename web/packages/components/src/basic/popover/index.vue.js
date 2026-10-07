import { cn } from '#utils';
import { Popover, PopoverContent, PopoverTrigger, } from './popover';
defineOptions({
    name: 'BuiltInPopover',
});
const props = defineProps();
function handleOpenAutoFocus(e) {
    e.preventDefault();
}
const open = defineModel('open', {
    default: false,
});
const __VLS_defaultModels = {
    'open': false,
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
/** @ts-ignore @type { | typeof __VLS_components.Popover | typeof __VLS_components.Popover} */
Popover;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    open: (__VLS_ctx.open),
}));
const __VLS_2 = __VLS_1({
    open: (__VLS_ctx.open),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.PopoverTrigger | typeof __VLS_components.PopoverTrigger} */
PopoverTrigger;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    asChild: true,
}));
const __VLS_9 = __VLS_8({
    asChild: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
var __VLS_13 = {};
// @ts-ignore
[open,];
var __VLS_10;
let __VLS_15;
/** @ts-ignore @type { | typeof __VLS_components.PopoverContent | typeof __VLS_components.PopoverContent} */
PopoverContent;
// @ts-ignore
const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({
    ...{ 'onOpenAutoFocus': {} },
    align: __VLS_ctx.align,
    alignOffset: __VLS_ctx.alignOffset,
    side: __VLS_ctx.side,
    sideOffset: __VLS_ctx.sideOffset,
    collisionPadding: __VLS_ctx.collisionPadding,
    ...{ class: (__VLS_ctx.cn('z-2000 w-unset min-w-72', props.class)) },
}));
const __VLS_17 = __VLS_16({
    ...{ 'onOpenAutoFocus': {} },
    align: __VLS_ctx.align,
    alignOffset: __VLS_ctx.alignOffset,
    side: __VLS_ctx.side,
    sideOffset: __VLS_ctx.sideOffset,
    collisionPadding: __VLS_ctx.collisionPadding,
    ...{ class: (__VLS_ctx.cn('z-2000 w-unset min-w-72', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_16));
let __VLS_20;
const __VLS_21 = {
    /** @type {typeof __VLS_20.openAutoFocus} */
    onOpenAutoFocus: (__VLS_ctx.handleOpenAutoFocus),
};
const { default: __VLS_22 } = __VLS_18.slots;
var __VLS_23 = {};
// @ts-ignore
[align, alignOffset, side, sideOffset, collisionPadding, cn, handleOpenAutoFocus,];
var __VLS_18;
var __VLS_19;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_14 = __VLS_13, __VLS_24 = __VLS_23;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
