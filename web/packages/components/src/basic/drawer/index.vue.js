import { VisuallyHidden } from 'reka-ui';
import { computed, nextTick, ref, shallowRef, useId, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import Button from '../button/index.vue';
import Icon from '../icon/index.vue';
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle, } from './sheet';
defineOptions({
    name: 'BuiltInDrawer',
});
const props = withDefaults(defineProps(), {
    modelValue: false,
    zIndex: 2000,
    side: 'right',
    loading: false,
    closable: true,
    centered: false,
    bordered: true,
    overlay: true,
    overlayBlur: false,
    showConfirmButton: true,
    showCancelButton: false,
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    confirmButtonDisabled: false,
    confirmButtonLoading: false,
    header: true,
    footer: true,
    closeOnClickOverlay: true,
    closeOnPressEscape: true,
    destroyOnClose: true,
    openAutoFocus: false,
});
const emits = defineEmits();
const slots = defineSlots();
const sheetContentRef = useTemplateRef({});
const drawerId = shallowRef(props.id ?? useId());
const isOpen = ref(props.modelValue);
watch(() => props.modelValue, (newValue) => {
    isOpen.value = newValue;
});
const hasOpened = ref(false);
const isClosed = ref(!props.modelValue);
watch(isOpen, (val) => {
    emits('update:modelValue', val);
    if (val) {
        emits('open');
    }
    else {
        emits('close');
    }
    isClosed.value = false;
    if (val && !hasOpened.value) {
        hasOpened.value = true;
    }
});
const forceMount = computed(() => !props.destroyOnClose && hasOpened.value);
async function updateOpen(value) {
    if (value) {
        isOpen.value = value;
        emits('open');
    }
    else {
        if (props.beforeClose) {
            await props.beforeClose('close', () => {
                isOpen.value = value;
                emits('close');
            });
        }
        else {
            isOpen.value = value;
            emits('close');
        }
    }
}
const isConfirmButtonLoading = ref(false);
async function onConfirm() {
    if (props.beforeClose) {
        isConfirmButtonLoading.value = true;
        await props.beforeClose('confirm', () => {
            isOpen.value = false;
            emits('confirm');
        });
        isConfirmButtonLoading.value = false;
    }
    else {
        isOpen.value = false;
        emits('confirm');
    }
}
async function onCancel() {
    if (props.beforeClose) {
        await props.beforeClose('cancel', () => {
            isOpen.value = false;
            emits('cancel');
        });
    }
    else {
        isOpen.value = false;
        emits('cancel');
    }
}
function handleOpenAutoFocus(e) {
    if (!props.openAutoFocus) {
        e.preventDefault();
        e.stopPropagation();
        nextTick(() => {
            sheetContentRef.value?.el?.$el?.focus();
        });
    }
}
function handleFocusOutside(e) {
    e.preventDefault();
    e.stopPropagation();
}
function handleClickOutside(e) {
    if (!props.closeOnClickOverlay || e.target.dataset.drawerId !== drawerId.value) {
        e.preventDefault();
        e.stopPropagation();
    }
}
function handleEscapeKeyDown(e) {
    if (!props.closeOnPressEscape) {
        e.preventDefault();
        e.stopPropagation();
    }
}
function handleAnimationEnd() {
    if (isOpen.value) {
        emits('opened');
    }
    else {
        emits('closed');
        isClosed.value = true;
    }
}
const __VLS_defaults = {
    modelValue: false,
    zIndex: 2000,
    side: 'right',
    loading: false,
    closable: true,
    centered: false,
    bordered: true,
    overlay: true,
    overlayBlur: false,
    showConfirmButton: true,
    showCancelButton: false,
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    confirmButtonDisabled: false,
    confirmButtonLoading: false,
    header: true,
    footer: true,
    closeOnClickOverlay: true,
    closeOnPressEscape: true,
    destroyOnClose: true,
    openAutoFocus: false,
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
/** @ts-ignore @type { | typeof __VLS_components.Sheet | typeof __VLS_components.Sheet} */
Sheet;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onUpdate:open': {} },
    modal: (false),
    open: (__VLS_ctx.isOpen),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onUpdate:open': {} },
    modal: (false),
    open: (__VLS_ctx.isOpen),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.'update:open'} */
    'onUpdate:open': (__VLS_ctx.updateOpen),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
let __VLS_9;
/** @ts-ignore @type { | typeof __VLS_components.SheetContent | typeof __VLS_components.SheetContent} */
SheetContent;
// @ts-ignore
const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
    ...{ 'onOpenAutoFocus': {} },
    ...{ 'onCloseAutoFocus': {} },
    ...{ 'onFocusOutside': {} },
    ...{ 'onPointerDownOutside': {} },
    ...{ 'onInteractOutside': {} },
    ...{ 'onEscapeKeyDown': {} },
    ...{ 'onAnimationEnd': {} },
    ref: "sheetContentRef",
    drawerId: (__VLS_ctx.drawerId),
    open: (__VLS_ctx.isOpen),
    zIndex: (props.zIndex),
    closable: (props.closable),
    overlay: (props.overlay),
    overlayBlur: (props.overlayBlur),
    ...{ class: (__VLS_ctx.cn('z-2000 w-full flex flex-col gap-0 p-0', props.contentClass, {
            hidden: __VLS_ctx.isClosed,
        })) },
    side: (props.side),
    forceMount: (__VLS_ctx.forceMount),
}));
const __VLS_11 = __VLS_10({
    ...{ 'onOpenAutoFocus': {} },
    ...{ 'onCloseAutoFocus': {} },
    ...{ 'onFocusOutside': {} },
    ...{ 'onPointerDownOutside': {} },
    ...{ 'onInteractOutside': {} },
    ...{ 'onEscapeKeyDown': {} },
    ...{ 'onAnimationEnd': {} },
    ref: "sheetContentRef",
    drawerId: (__VLS_ctx.drawerId),
    open: (__VLS_ctx.isOpen),
    zIndex: (props.zIndex),
    closable: (props.closable),
    overlay: (props.overlay),
    overlayBlur: (props.overlayBlur),
    ...{ class: (__VLS_ctx.cn('z-2000 w-full flex flex-col gap-0 p-0', props.contentClass, {
            hidden: __VLS_ctx.isClosed,
        })) },
    side: (props.side),
    forceMount: (__VLS_ctx.forceMount),
}, ...__VLS_functionalComponentArgsRest(__VLS_10));
let __VLS_14;
const __VLS_15 = {
    /** @type {typeof __VLS_14.openAutoFocus} */
    onOpenAutoFocus: (__VLS_ctx.handleOpenAutoFocus),
};
const __VLS_16 = {
    /** @type {typeof __VLS_14.closeAutoFocus} */
    onCloseAutoFocus: (__VLS_ctx.handleFocusOutside),
};
const __VLS_17 = {
    /** @type {typeof __VLS_14.focusOutside} */
    onFocusOutside: (__VLS_ctx.handleFocusOutside),
};
const __VLS_18 = {
    /** @type {typeof __VLS_14.pointerDownOutside} */
    onPointerDownOutside: (__VLS_ctx.handleClickOutside),
};
const __VLS_19 = {
    /** @type {typeof __VLS_14.interactOutside} */
    onInteractOutside: (__VLS_ctx.handleClickOutside),
};
const __VLS_20 = {
    /** @type {typeof __VLS_14.escapeKeyDown} */
    onEscapeKeyDown: (__VLS_ctx.handleEscapeKeyDown),
};
const __VLS_21 = {
    /** @type {typeof __VLS_14.animationEnd} */
    onAnimationEnd: (__VLS_ctx.handleAnimationEnd),
};
var __VLS_22;
const { default: __VLS_24 } = __VLS_12.slots;
if (__VLS_ctx.header) {
    let __VLS_25;
    /** @ts-ignore @type { | typeof __VLS_components.SheetHeader | typeof __VLS_components.SheetHeader} */
    SheetHeader;
    // @ts-ignore
    const __VLS_26 = __VLS_asFunctionalComponent1(__VLS_25, new __VLS_25({
        ...{ class: (__VLS_ctx.cn('p-4 gap-y-1', props.headerClass, {
                'border-b': props.bordered,
            })) },
    }));
    const __VLS_27 = __VLS_26({
        ...{ class: (__VLS_ctx.cn('p-4 gap-y-1', props.headerClass, {
                'border-b': props.bordered,
            })) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_26));
    const { default: __VLS_30 } = __VLS_28.slots;
    if (!!slots.header) {
        let __VLS_31;
        /** @ts-ignore @type { | typeof __VLS_components.VisuallyHidden | typeof __VLS_components.VisuallyHidden} */
        VisuallyHidden;
        // @ts-ignore
        const __VLS_32 = __VLS_asFunctionalComponent1(__VLS_31, new __VLS_31({}));
        const __VLS_33 = __VLS_32({}, ...__VLS_functionalComponentArgsRest(__VLS_32));
        const { default: __VLS_36 } = __VLS_34.slots;
        let __VLS_37;
        /** @ts-ignore @type { | typeof __VLS_components.SheetTitle} */
        SheetTitle;
        // @ts-ignore
        const __VLS_38 = __VLS_asFunctionalComponent1(__VLS_37, new __VLS_37({}));
        const __VLS_39 = __VLS_38({}, ...__VLS_functionalComponentArgsRest(__VLS_38));
        let __VLS_42;
        /** @ts-ignore @type { | typeof __VLS_components.SheetDescription} */
        SheetDescription;
        // @ts-ignore
        const __VLS_43 = __VLS_asFunctionalComponent1(__VLS_42, new __VLS_42({}));
        const __VLS_44 = __VLS_43({}, ...__VLS_functionalComponentArgsRest(__VLS_43));
        // @ts-ignore
        [isOpen, isOpen, updateOpen, drawerId, cn, cn, isClosed, forceMount, handleOpenAutoFocus, handleFocusOutside, handleFocusOutside, handleClickOutside, handleClickOutside, handleEscapeKeyDown, handleAnimationEnd, header,];
        var __VLS_34;
    }
    __VLS_asFunctionalSlot(slots.header)({});
    let __VLS_48;
    /** @ts-ignore @type { | typeof __VLS_components.SheetTitle | typeof __VLS_components.SheetTitle} */
    SheetTitle;
    // @ts-ignore
    const __VLS_49 = __VLS_asFunctionalComponent1(__VLS_48, new __VLS_48({
        ...{ class: ({ 'text-center': props.centered }) },
    }));
    const __VLS_50 = __VLS_49({
        ...{ class: ({ 'text-center': props.centered }) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_49));
    /** @type {__VLS_StyleScopedClasses['text-center']} */ ;
    const { default: __VLS_53 } = __VLS_51.slots;
    (typeof props.title === 'function' ? props.title() : props.title);
    // @ts-ignore
    [];
    var __VLS_51;
    let __VLS_54;
    /** @ts-ignore @type { | typeof __VLS_components.SheetDescription | typeof __VLS_components.SheetDescription} */
    SheetDescription;
    // @ts-ignore
    const __VLS_55 = __VLS_asFunctionalComponent1(__VLS_54, new __VLS_54({
        ...{ class: "empty:hidden" },
        ...{ class: ({ 'text-center': props.centered }) },
    }));
    const __VLS_56 = __VLS_55({
        ...{ class: "empty:hidden" },
        ...{ class: ({ 'text-center': props.centered }) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_55));
    /** @type {__VLS_StyleScopedClasses['empty:hidden']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-center']} */ ;
    const { default: __VLS_59 } = __VLS_57.slots;
    (typeof props.description === 'function' ? props.description() : props.description);
    // @ts-ignore
    [];
    var __VLS_57;
    // @ts-ignore
    [];
    var __VLS_28;
}
else {
    let __VLS_60;
    /** @ts-ignore @type { | typeof __VLS_components.VisuallyHidden | typeof __VLS_components.VisuallyHidden} */
    VisuallyHidden;
    // @ts-ignore
    const __VLS_61 = __VLS_asFunctionalComponent1(__VLS_60, new __VLS_60({}));
    const __VLS_62 = __VLS_61({}, ...__VLS_functionalComponentArgsRest(__VLS_61));
    const { default: __VLS_65 } = __VLS_63.slots;
    let __VLS_66;
    /** @ts-ignore @type { | typeof __VLS_components.SheetTitle} */
    SheetTitle;
    // @ts-ignore
    const __VLS_67 = __VLS_asFunctionalComponent1(__VLS_66, new __VLS_66({}));
    const __VLS_68 = __VLS_67({}, ...__VLS_functionalComponentArgsRest(__VLS_67));
    let __VLS_71;
    /** @ts-ignore @type { | typeof __VLS_components.SheetDescription} */
    SheetDescription;
    // @ts-ignore
    const __VLS_72 = __VLS_asFunctionalComponent1(__VLS_71, new __VLS_71({}));
    const __VLS_73 = __VLS_72({}, ...__VLS_functionalComponentArgsRest(__VLS_72));
    // @ts-ignore
    [];
    var __VLS_63;
}
if (!!slots.default) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "m-0 p-4 flex-1 relative overflow-y-auto" },
    });
    /** @type {__VLS_StyleScopedClasses['m-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['p-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
    /** @type {__VLS_StyleScopedClasses['relative']} */ ;
    /** @type {__VLS_StyleScopedClasses['overflow-y-auto']} */ ;
    __VLS_asFunctionalSlot(slots['default'])({});
}
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "bg-popover/75 flex-center size-full inset-0 absolute z-1000" },
});
__VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (props.loading), }, null, null);
/** @type {__VLS_StyleScopedClasses['bg-popover/75']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
/** @type {__VLS_StyleScopedClasses['size-full']} */ ;
/** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
/** @type {__VLS_StyleScopedClasses['absolute']} */ ;
/** @type {__VLS_StyleScopedClasses['z-1000']} */ ;
const __VLS_77 = Icon;
// @ts-ignore
const __VLS_78 = __VLS_asFunctionalComponent1(__VLS_77, new __VLS_77({
    name: "i-line-md:loading-twotone-loop",
    ...{ class: "size-10" },
}));
const __VLS_79 = __VLS_78({
    name: "i-line-md:loading-twotone-loop",
    ...{ class: "size-10" },
}, ...__VLS_functionalComponentArgsRest(__VLS_78));
/** @type {__VLS_StyleScopedClasses['size-10']} */ ;
if (__VLS_ctx.footer) {
    let __VLS_82;
    /** @ts-ignore @type { | typeof __VLS_components.SheetFooter | typeof __VLS_components.SheetFooter} */
    SheetFooter;
    // @ts-ignore
    const __VLS_83 = __VLS_asFunctionalComponent1(__VLS_82, new __VLS_82({
        ...{ class: (__VLS_ctx.cn('p-3 gap-y-2', props.footerClass, {
                'sm:justify-center': props.centered,
                'border-t': props.bordered,
            })) },
    }));
    const __VLS_84 = __VLS_83({
        ...{ class: (__VLS_ctx.cn('p-3 gap-y-2', props.footerClass, {
                'sm:justify-center': props.centered,
                'border-t': props.bordered,
            })) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_83));
    const { default: __VLS_87 } = __VLS_85.slots;
    __VLS_asFunctionalSlot(slots.footer)({});
    if (__VLS_ctx.showCancelButton) {
        const __VLS_89 = Button || Button;
        // @ts-ignore
        const __VLS_90 = __VLS_asFunctionalComponent1(__VLS_89, new __VLS_89({
            ...{ 'onClick': {} },
            variant: "outline",
        }));
        const __VLS_91 = __VLS_90({
            ...{ 'onClick': {} },
            variant: "outline",
        }, ...__VLS_functionalComponentArgsRest(__VLS_90));
        let __VLS_94;
        const __VLS_95 = {
            /** @type {typeof __VLS_94.click} */
            onClick: (__VLS_ctx.onCancel),
        };
        const { default: __VLS_96 } = __VLS_92.slots;
        (typeof props.cancelButtonText === 'function' ? props.cancelButtonText() : props.cancelButtonText);
        // @ts-ignore
        [cn, footer, showCancelButton, onCancel,];
        var __VLS_92;
        var __VLS_93;
    }
    if (__VLS_ctx.showConfirmButton) {
        const __VLS_97 = Button || Button;
        // @ts-ignore
        const __VLS_98 = __VLS_asFunctionalComponent1(__VLS_97, new __VLS_97({
            ...{ 'onClick': {} },
            disabled: (__VLS_ctx.confirmButtonDisabled),
            loading: (__VLS_ctx.confirmButtonLoading),
        }));
        const __VLS_99 = __VLS_98({
            ...{ 'onClick': {} },
            disabled: (__VLS_ctx.confirmButtonDisabled),
            loading: (__VLS_ctx.confirmButtonLoading),
        }, ...__VLS_functionalComponentArgsRest(__VLS_98));
        let __VLS_102;
        const __VLS_103 = {
            /** @type {typeof __VLS_102.click} */
            onClick: (__VLS_ctx.onConfirm),
        };
        const { default: __VLS_104 } = __VLS_100.slots;
        (typeof props.confirmButtonText === 'function' ? props.confirmButtonText() : props.confirmButtonText);
        // @ts-ignore
        [showConfirmButton, confirmButtonDisabled, confirmButtonLoading, onConfirm,];
        var __VLS_100;
        var __VLS_101;
    }
    // @ts-ignore
    [];
    var __VLS_85;
}
// @ts-ignore
[];
var __VLS_12;
var __VLS_13;
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
// @ts-ignore
var __VLS_23 = __VLS_22;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
