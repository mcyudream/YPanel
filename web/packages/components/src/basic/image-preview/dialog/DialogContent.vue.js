import { X } from '@lucide/vue';
import { reactiveOmit } from '@vueuse/core';
import { DialogClose, DialogContent, DialogPortal, useForwardPropsEmits, } from 'reka-ui';
import { cn } from '#utils';
import DialogOverlay from './DialogOverlay.vue';
defineOptions({
    inheritAttrs: false,
});
const props = withDefaults(defineProps(), {
    showCloseButton: true,
});
const emits = defineEmits();
const delegatedProps = reactiveOmit(props, 'class');
const forwarded = useForwardPropsEmits(delegatedProps, emits);
const __VLS_defaults = {
    showCloseButton: true,
};
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
/** @ts-ignore @type { | typeof __VLS_components.DialogPortal | typeof __VLS_components.DialogPortal} */
DialogPortal;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({}));
const __VLS_2 = __VLS_1({}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
const __VLS_7 = DialogOverlay;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    ...{ class: "z-2000" },
}));
const __VLS_9 = __VLS_8({
    ...{ class: "z-2000" },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
/** @type {__VLS_StyleScopedClasses['z-2000']} */ ;
let __VLS_12;
/** @ts-ignore @type { | typeof __VLS_components.DialogContent | typeof __VLS_components.DialogContent} */
DialogContent;
// @ts-ignore
const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
    ...{ 'onAnimationend': {} },
    dataSlot: "dialog-content",
    ...({ ...__VLS_ctx.$attrs, ...__VLS_ctx.forwarded }),
    ...{ class: (__VLS_ctx.cn('data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed inset-0 z-2000 size-full duration-200', props.class)) },
}));
const __VLS_14 = __VLS_13({
    ...{ 'onAnimationend': {} },
    dataSlot: "dialog-content",
    ...({ ...__VLS_ctx.$attrs, ...__VLS_ctx.forwarded }),
    ...{ class: (__VLS_ctx.cn('data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed inset-0 z-2000 size-full duration-200', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_13));
let __VLS_17;
const __VLS_18 = {
    /** @type {typeof __VLS_17.animationend} */
    onAnimationend: (...[$event]) => {
        return (__VLS_ctx.emits('animationEnd'));
        // @ts-ignore
        [$attrs, forwarded, cn, emits,];
    },
};
const { default: __VLS_19 } = __VLS_15.slots;
var __VLS_20 = {};
if (__VLS_ctx.showCloseButton) {
    let __VLS_22;
    /** @ts-ignore @type { | typeof __VLS_components.DialogClose | typeof __VLS_components.DialogClose} */
    DialogClose;
    // @ts-ignore
    const __VLS_23 = __VLS_asFunctionalComponent1(__VLS_22, new __VLS_22({
        dataSlot: "dialog-close",
        ...{ class: "rounded-xs bg-background opacity-70 ring-offset-background transition-opacity right-4 top-4 absolute data-[state=open]:text-muted-foreground focus:outline-hidden data-[state=open]:bg-accent hover:opacity-100 [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-6 [&_svg]:pointer-events-none disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    }));
    const __VLS_24 = __VLS_23({
        dataSlot: "dialog-close",
        ...{ class: "rounded-xs bg-background opacity-70 ring-offset-background transition-opacity right-4 top-4 absolute data-[state=open]:text-muted-foreground focus:outline-hidden data-[state=open]:bg-accent hover:opacity-100 [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-6 [&_svg]:pointer-events-none disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_23));
    /** @type {__VLS_StyleScopedClasses['rounded-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-background']} */ ;
    /** @type {__VLS_StyleScopedClasses['opacity-70']} */ ;
    /** @type {__VLS_StyleScopedClasses['ring-offset-background']} */ ;
    /** @type {__VLS_StyleScopedClasses['transition-opacity']} */ ;
    /** @type {__VLS_StyleScopedClasses['right-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:outline-hidden']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:bg-accent']} */ ;
    /** @type {__VLS_StyleScopedClasses['hover:opacity-100']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg]:shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg:not([class*=size-])]:size-6']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg]:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['disabled:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-ring']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-offset-2']} */ ;
    const { default: __VLS_27 } = __VLS_25.slots;
    let __VLS_28;
    /** @ts-ignore @type { | typeof __VLS_components.X} */
    X;
    // @ts-ignore
    const __VLS_29 = __VLS_asFunctionalComponent1(__VLS_28, new __VLS_28({}));
    const __VLS_30 = __VLS_29({}, ...__VLS_functionalComponentArgsRest(__VLS_29));
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "sr-only" },
    });
    /** @type {__VLS_StyleScopedClasses['sr-only']} */ ;
    // @ts-ignore
    [showCloseButton,];
    var __VLS_25;
}
// @ts-ignore
[];
var __VLS_15;
var __VLS_16;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_21 = __VLS_20;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
