import { Maximize, Minimize, X } from '@lucide/vue';
import { reactiveOmit, useScrollLock } from '@vueuse/core';
import { DialogClose, DialogContent, DialogPortal, useForwardPropsEmits, } from 'reka-ui';
import { computed, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import DialogOverlay from './DialogOverlay.vue';
defineOptions({
    inheritAttrs: false,
});
const props = defineProps();
const emits = defineEmits();
const delegatedProps = reactiveOmit(props, 'class');
const forwarded = useForwardPropsEmits(delegatedProps, emits);
function handleMaximize() {
    emits('toggleMaximize', !props.maximize);
}
const dialogContentRef = useTemplateRef('dialogContentRef');
const __VLS_exposed = {
    el: dialogContentRef,
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
        'data-modal-id': (props.modalId),
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
const __VLS_13 = DialogOverlay;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({}));
const __VLS_15 = __VLS_14({}, ...__VLS_functionalComponentArgsRest(__VLS_14));
let __VLS_18;
/** @ts-ignore @type { | typeof __VLS_components.DialogContent | typeof __VLS_components.DialogContent} */
DialogContent;
// @ts-ignore
const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
    ...{ 'onAnimationend': {} },
    ref: "dialogContentRef",
    ...(__VLS_ctx.forwarded),
    ...{ class: (__VLS_ctx.cn('bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 rounded-lg border p-6 shadow-lg duration-200 sm:max-w-lg outline-hidden', props.class)) },
    ...{ style: ({
            zIndex: props.zIndex,
        }) },
}));
const __VLS_20 = __VLS_19({
    ...{ 'onAnimationend': {} },
    ref: "dialogContentRef",
    ...(__VLS_ctx.forwarded),
    ...{ class: (__VLS_ctx.cn('bg-background data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 fixed top-[50%] left-[50%] z-50 grid w-full max-w-[calc(100%-2rem)] translate-x-[-50%] translate-y-[-50%] gap-4 rounded-lg border p-6 shadow-lg duration-200 sm:max-w-lg outline-hidden', props.class)) },
    ...{ style: ({
            zIndex: props.zIndex,
        }) },
}, ...__VLS_functionalComponentArgsRest(__VLS_19));
let __VLS_23;
const __VLS_24 = {
    /** @type {typeof __VLS_23.animationend} */
    onAnimationend: (...[$event]) => {
        return (__VLS_ctx.emits('animationEnd'));
        // @ts-ignore
        [cn, forwarded, emits,];
    },
};
var __VLS_25;
const { default: __VLS_27 } = __VLS_21.slots;
var __VLS_28 = {};
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex-center gap-2 inset-e-4 inset-t-4 absolute" },
});
/** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
/** @type {__VLS_StyleScopedClasses['inset-e-4']} */ ;
/** @type {__VLS_StyleScopedClasses['inset-t-4']} */ ;
/** @type {__VLS_StyleScopedClasses['absolute']} */ ;
if (props.maximizable) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (__VLS_ctx.handleMaximize) },
        ...{ class: "rounded-xs opacity-70 ring-offset-background transition-opacity right-4 top-4 data-[state=open]:text-muted-foreground focus:outline-hidden data-[state=open]:bg-accent hover:opacity-100 [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4 [&_svg]:pointer-events-none disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    });
    /** @type {__VLS_StyleScopedClasses['rounded-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['opacity-70']} */ ;
    /** @type {__VLS_StyleScopedClasses['ring-offset-background']} */ ;
    /** @type {__VLS_StyleScopedClasses['transition-opacity']} */ ;
    /** @type {__VLS_StyleScopedClasses['right-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:outline-hidden']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:bg-accent']} */ ;
    /** @type {__VLS_StyleScopedClasses['hover:opacity-100']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg]:shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg:not([class*=size-])]:size-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg]:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['disabled:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-ring']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-offset-2']} */ ;
    if (!props.maximize) {
        let __VLS_30;
        /** @ts-ignore @type { | typeof __VLS_components.Maximize} */
        Maximize;
        // @ts-ignore
        const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({}));
        const __VLS_32 = __VLS_31({}, ...__VLS_functionalComponentArgsRest(__VLS_31));
    }
    else {
        let __VLS_35;
        /** @ts-ignore @type { | typeof __VLS_components.Minimize} */
        Minimize;
        // @ts-ignore
        const __VLS_36 = __VLS_asFunctionalComponent1(__VLS_35, new __VLS_35({}));
        const __VLS_37 = __VLS_36({}, ...__VLS_functionalComponentArgsRest(__VLS_36));
    }
}
if (__VLS_ctx.closable) {
    let __VLS_40;
    /** @ts-ignore @type { | typeof __VLS_components.DialogClose | typeof __VLS_components.DialogClose} */
    DialogClose;
    // @ts-ignore
    const __VLS_41 = __VLS_asFunctionalComponent1(__VLS_40, new __VLS_40({
        ...{ class: "rounded-xs opacity-70 ring-offset-background transition-opacity right-4 top-4 data-[state=open]:text-muted-foreground focus:outline-hidden data-[state=open]:bg-accent hover:opacity-100 [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4 [&_svg]:pointer-events-none disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    }));
    const __VLS_42 = __VLS_41({
        ...{ class: "rounded-xs opacity-70 ring-offset-background transition-opacity right-4 top-4 data-[state=open]:text-muted-foreground focus:outline-hidden data-[state=open]:bg-accent hover:opacity-100 [&_svg]:shrink-0 [&_svg:not([class*=size-])]:size-4 [&_svg]:pointer-events-none disabled:pointer-events-none focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_41));
    /** @type {__VLS_StyleScopedClasses['rounded-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['opacity-70']} */ ;
    /** @type {__VLS_StyleScopedClasses['ring-offset-background']} */ ;
    /** @type {__VLS_StyleScopedClasses['transition-opacity']} */ ;
    /** @type {__VLS_StyleScopedClasses['right-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:outline-hidden']} */ ;
    /** @type {__VLS_StyleScopedClasses['data-[state=open]:bg-accent']} */ ;
    /** @type {__VLS_StyleScopedClasses['hover:opacity-100']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg]:shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg:not([class*=size-])]:size-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['[&_svg]:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['disabled:pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-ring']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-offset-2']} */ ;
    const { default: __VLS_45 } = __VLS_43.slots;
    let __VLS_46;
    /** @ts-ignore @type { | typeof __VLS_components.X} */
    X;
    // @ts-ignore
    const __VLS_47 = __VLS_asFunctionalComponent1(__VLS_46, new __VLS_46({}));
    const __VLS_48 = __VLS_47({}, ...__VLS_functionalComponentArgsRest(__VLS_47));
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "sr-only" },
    });
    /** @type {__VLS_StyleScopedClasses['sr-only']} */ ;
    // @ts-ignore
    [handleMaximize, closable,];
    var __VLS_43;
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
});
const __VLS_export = {};
export default {};
