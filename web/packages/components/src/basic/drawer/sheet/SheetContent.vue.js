import { X } from '@lucide/vue';
import { reactiveOmit, useScrollLock } from '@vueuse/core';
import { DialogClose, DialogContent, DialogPortal, useForwardPropsEmits, } from 'reka-ui';
import { computed, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import SheetOverlay from './SheetOverlay.vue';
defineOptions({
    inheritAttrs: false,
});
const props = withDefaults(defineProps(), {
    side: 'right',
});
const emits = defineEmits();
const delegatedProps = reactiveOmit(props, 'class', 'side');
const forwarded = useForwardPropsEmits(delegatedProps, emits);
const sheetContentRef = useTemplateRef('sheetContentRef');
const __VLS_exposed = {
    el: sheetContentRef,
};
defineExpose(__VLS_exposed);
const showOverlay = computed(() => props.open && props.overlay);
const isLocked = useScrollLock(document.body);
watch(showOverlay, (val) => {
    if (val) {
        isLocked.value = true;
    }
    else {
        isLocked.value = false;
    }
});
const __VLS_defaults = {
    side: 'right',
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
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.Transition | typeof __VLS_components.Transition} */
Transition;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    ...({
        enterActiveClass: 'ease-in-out duration-300',
        enterFromClass: 'opacity-0',
        enterToClass: 'opacity-100',
        leaveActiveClass: 'ease-in-out duration-300',
        leaveFromClass: 'opacity-100',
        leaveToClass: 'opacity-0',
    }),
    appear: (true),
}));
const __VLS_9 = __VLS_8({
    ...({
        enterActiveClass: 'ease-in-out duration-300',
        enterFromClass: 'opacity-0',
        enterToClass: 'opacity-100',
        leaveActiveClass: 'ease-in-out duration-300',
        leaveFromClass: 'opacity-100',
        leaveToClass: 'opacity-0',
    }),
    appear: (true),
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
if (__VLS_ctx.showOverlay) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div)({
        'data-drawer-id': (props.drawerId),
        ...{ class: (__VLS_ctx.cn('fixed inset-0 pointer-events-auto data-[state=closed]:animate-out data-[state=open]:animate-in bg-black/50 data-[state=open]:fade-in-0 data-[state=closed]:fade-out-0', {
                'backdrop-blur-sm': props.overlayBlur,
            })) },
        ...{ style: ({
                zIndex: props.zIndex,
            }) },
    });
}
// @ts-ignore
[showOverlay, cn,];
var __VLS_10;
const __VLS_13 = SheetOverlay;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({}));
const __VLS_15 = __VLS_14({}, ...__VLS_functionalComponentArgsRest(__VLS_14));
let __VLS_18;
/** @ts-ignore @type { | typeof __VLS_components.DialogContent | typeof __VLS_components.DialogContent} */
DialogContent;
// @ts-ignore
const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
    ...{ 'onAnimationend': {} },
    ref: "sheetContentRef",
    dataSlot: "sheet-content",
    ...{ class: (__VLS_ctx.cn('bg-background data-[state=open]:animate-in data-[state=closed]:animate-out fixed z-50 flex flex-col gap-4 shadow-lg transition ease-in-out data-[state=closed]:duration-300 data-[state=open]:duration-500', __VLS_ctx.side === 'right'
            && 'data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm', __VLS_ctx.side === 'left'
            && 'data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left inset-y-0 left-0 h-full w-3/4 border-r sm:max-w-sm', __VLS_ctx.side === 'top'
            && 'data-[state=closed]:slide-out-to-top data-[state=open]:slide-in-from-top inset-x-0 top-0 h-auto border-b', __VLS_ctx.side === 'bottom'
            && 'data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom inset-x-0 bottom-0 h-auto border-t', props.class)) },
    ...{ style: ({
            zIndex: props.zIndex,
        }) },
    ...({ ...__VLS_ctx.$attrs, ...__VLS_ctx.forwarded }),
}));
const __VLS_20 = __VLS_19({
    ...{ 'onAnimationend': {} },
    ref: "sheetContentRef",
    dataSlot: "sheet-content",
    ...{ class: (__VLS_ctx.cn('bg-background data-[state=open]:animate-in data-[state=closed]:animate-out fixed z-50 flex flex-col gap-4 shadow-lg transition ease-in-out data-[state=closed]:duration-300 data-[state=open]:duration-500', __VLS_ctx.side === 'right'
            && 'data-[state=closed]:slide-out-to-right data-[state=open]:slide-in-from-right inset-y-0 right-0 h-full w-3/4 border-l sm:max-w-sm', __VLS_ctx.side === 'left'
            && 'data-[state=closed]:slide-out-to-left data-[state=open]:slide-in-from-left inset-y-0 left-0 h-full w-3/4 border-r sm:max-w-sm', __VLS_ctx.side === 'top'
            && 'data-[state=closed]:slide-out-to-top data-[state=open]:slide-in-from-top inset-x-0 top-0 h-auto border-b', __VLS_ctx.side === 'bottom'
            && 'data-[state=closed]:slide-out-to-bottom data-[state=open]:slide-in-from-bottom inset-x-0 bottom-0 h-auto border-t', props.class)) },
    ...{ style: ({
            zIndex: props.zIndex,
        }) },
    ...({ ...__VLS_ctx.$attrs, ...__VLS_ctx.forwarded }),
}, ...__VLS_functionalComponentArgsRest(__VLS_19));
let __VLS_23;
const __VLS_24 = {
    /** @type {typeof __VLS_23.animationend} */
    onAnimationend: (...[$event]) => {
        return (__VLS_ctx.emits('animationEnd'));
        // @ts-ignore
        [cn, side, side, side, side, $attrs, forwarded, emits,];
    },
};
var __VLS_25;
const { default: __VLS_27 } = __VLS_21.slots;
var __VLS_28 = {};
if (__VLS_ctx.closable) {
    let __VLS_30;
    /** @ts-ignore @type { | typeof __VLS_components.DialogClose | typeof __VLS_components.DialogClose} */
    DialogClose;
    // @ts-ignore
    const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({
        ...{ class: "rounded-xs opacity-70 ring-offset-background transition-opacity right-4 top-4 absolute focus:outline-hidden data-[state=open]:bg-secondary hover:opacity-100 disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    }));
    const __VLS_32 = __VLS_31({
        ...{ class: "rounded-xs opacity-70 ring-offset-background transition-opacity right-4 top-4 absolute focus:outline-hidden data-[state=open]:bg-secondary hover:opacity-100 disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_31));
    /** @type {__VLS_StyleScopedClasses['rounded-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['opacity-70']} */ ;
    /** @type {__VLS_StyleScopedClasses['ring-offset-background']} */ ;
    /** @type {__VLS_StyleScopedClasses['transition-opacity']} */ ;
    /** @type {__VLS_StyleScopedClasses['right-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:outline-hidden']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:bg-secondary']} */ ;
    /** @type {__VLS_StyleScopedClasses['hover:opacity-100']} */ ;
    /** @type {__VLS_StyleScopedClasses['disabled:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-ring']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-offset-2']} */ ;
    const { default: __VLS_35 } = __VLS_33.slots;
    let __VLS_36;
    /** @ts-ignore @type { | typeof __VLS_components.X} */
    X;
    // @ts-ignore
    const __VLS_37 = __VLS_asFunctionalComponent1(__VLS_36, new __VLS_36({
        ...{ class: "size-4" },
    }));
    const __VLS_38 = __VLS_37({
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_37));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "sr-only" },
    });
    /** @type {__VLS_StyleScopedClasses['sr-only']} */ ;
    // @ts-ignore
    [closable,];
    var __VLS_33;
}
// @ts-ignore
[];
var __VLS_21;
var __VLS_22;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_26 = __VLS_25, __VLS_29 = __VLS_28;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    setup: () => __VLS_exposed,
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
