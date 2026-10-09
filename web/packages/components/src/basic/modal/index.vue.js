import { VisuallyHidden } from 'reka-ui';
import { computed, inject, nextTick, ref, shallowRef, useId, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import Button from '../button/index.vue';
import Icon from '../icon/index.vue';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, } from './dialog';
import { useDraggable } from './use-draggable';
defineOptions({
    name: 'BuiltInModal',
});
const props = withDefaults(defineProps(), {
    modelValue: false,
    zIndex: 2000,
    loading: false,
    closable: true,
    maximize: false,
    maximizable: false,
    draggable: false,
    center: false,
    border: true,
    alignCenter: false,
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
/** 弹窗挂载容器注入键（宿主可 provide 聚焦窗口的 body 实现窗口内模态） */
const injectedContainer = inject('fa:modal-container', undefined);
const portalTarget = computed(() => typeof injectedContainer?.value === 'object' || typeof injectedContainer?.value === 'string'
    ? injectedContainer.value
    : undefined);
const slots = defineSlots();
const dialogContentRef = useTemplateRef({});
const dialogHeaderRef = ref();
const dialogRef = ref();
const modalId = shallowRef(props.id ?? useId());
const isOpen = ref(props.modelValue);
const isMaximize = ref(props.maximize);
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
const { isDragging, transform } = useDraggable(dialogRef, dialogHeaderRef, computed(() => props.draggable && props.header && !isMaximize.value));
function setTransform() {
    if (isMaximize.value) {
        dialogRef.value.style.transform = 'none';
    }
    else {
        dialogRef.value.style.transform = `translate(${transform.offsetX}px, ${transform.offsetY}px)`;
    }
}
watch(isOpen, (val) => {
    if (val) {
        nextTick(() => {
            if (dialogContentRef.value) {
                dialogRef.value = dialogContentRef.value.el?.$el;
                setTransform();
            }
        });
    }
});
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
            dialogContentRef.value?.el?.$el?.focus();
        });
    }
}
function handleFocusOutside(e) {
    e.preventDefault();
    e.stopPropagation();
}
function handleClickOutside(e) {
    if (!props.closeOnClickOverlay || e.target.dataset.modalId !== modalId.value) {
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
function handleMaximize(val) {
    isMaximize.value = val;
    setTransform();
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
    loading: false,
    closable: true,
    maximize: false,
    maximizable: false,
    draggable: false,
    center: false,
    border: true,
    alignCenter: false,
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
/** @ts-ignore @type { | typeof __VLS_components.Dialog | typeof __VLS_components.Dialog} */
Dialog;
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
/** @ts-ignore @type { | typeof __VLS_components.DialogContent | typeof __VLS_components.DialogContent} */
DialogContent;
// @ts-ignore
const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
    ...{ 'onOpenAutoFocus': {} },
    ...{ 'onCloseAutoFocus': {} },
    ...{ 'onFocusOutside': {} },
    ...{ 'onPointerDownOutside': {} },
    ...{ 'onInteractOutside': {} },
    ...{ 'onEscapeKeyDown': {} },
    ...{ 'onToggleMaximize': {} },
    ...{ 'onAnimationEnd': {} },
    ref: "dialogContentRef",
    modalId: (__VLS_ctx.modalId),
    portalTo: (__VLS_ctx.portalTarget),
    open: (__VLS_ctx.isOpen),
    zIndex: (props.zIndex),
    closable: (props.closable),
    overlay: (props.overlay),
    overlayBlur: (props.overlayBlur),
    maximize: (__VLS_ctx.isMaximize),
    maximizable: (props.maximizable),
    forceMount: (__VLS_ctx.forceMount),
    ...{ class: (__VLS_ctx.cn('z-2000 top-0 sm:top-[5vh] translate-y-0 flex flex-col p-0 gap-0 mx-auto overflow-hidden max-w-full h-[calc-size(auto,size)] min-h-full max-h-full sm:min-h-auto sm:max-h-[90vh]', props.class, {
            'sm:top-0 size-full max-w-full sm:max-w-full max-h-full sm:max-h-full': __VLS_ctx.isMaximize,
            'top-1/2 -translate-y-1/2 min-h-auto sm:top-1/2 sm:-translate-y-1/2': props.alignCenter,
            'duration-0': __VLS_ctx.isDragging,
            'hidden': __VLS_ctx.isClosed,
        })) },
}));
const __VLS_11 = __VLS_10({
    ...{ 'onOpenAutoFocus': {} },
    ...{ 'onCloseAutoFocus': {} },
    ...{ 'onFocusOutside': {} },
    ...{ 'onPointerDownOutside': {} },
    ...{ 'onInteractOutside': {} },
    ...{ 'onEscapeKeyDown': {} },
    ...{ 'onToggleMaximize': {} },
    ...{ 'onAnimationEnd': {} },
    ref: "dialogContentRef",
    modalId: (__VLS_ctx.modalId),
    portalTo: (__VLS_ctx.portalTarget),
    open: (__VLS_ctx.isOpen),
    zIndex: (props.zIndex),
    closable: (props.closable),
    overlay: (props.overlay),
    overlayBlur: (props.overlayBlur),
    maximize: (__VLS_ctx.isMaximize),
    maximizable: (props.maximizable),
    forceMount: (__VLS_ctx.forceMount),
    ...{ class: (__VLS_ctx.cn('z-2000 top-0 sm:top-[5vh] translate-y-0 flex flex-col p-0 gap-0 mx-auto overflow-hidden max-w-full h-[calc-size(auto,size)] min-h-full max-h-full sm:min-h-auto sm:max-h-[90vh]', props.class, {
            'sm:top-0 size-full max-w-full sm:max-w-full max-h-full sm:max-h-full': __VLS_ctx.isMaximize,
            'top-1/2 -translate-y-1/2 min-h-auto sm:top-1/2 sm:-translate-y-1/2': props.alignCenter,
            'duration-0': __VLS_ctx.isDragging,
            'hidden': __VLS_ctx.isClosed,
        })) },
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
    /** @type {typeof __VLS_14.toggleMaximize} */
    onToggleMaximize: (__VLS_ctx.handleMaximize),
};
const __VLS_22 = {
    /** @type {typeof __VLS_14.animationEnd} */
    onAnimationEnd: (__VLS_ctx.handleAnimationEnd),
};
var __VLS_23;
const { default: __VLS_25 } = __VLS_12.slots;
if (__VLS_ctx.header) {
    let __VLS_26;
    /** @ts-ignore @type { | typeof __VLS_components.DialogHeader | typeof __VLS_components.DialogHeader} */
    DialogHeader;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        ref: "dialogHeaderRef",
        ...{ class: (__VLS_ctx.cn('p-4', props.headerClass, {
                'cursor-move select-none': props.draggable,
                'border-b': props.border,
            })) },
    }));
    const __VLS_28 = __VLS_27({
        ref: "dialogHeaderRef",
        ...{ class: (__VLS_ctx.cn('p-4', props.headerClass, {
                'cursor-move select-none': props.draggable,
                'border-b': props.border,
            })) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    var __VLS_31;
    const { default: __VLS_33 } = __VLS_29.slots;
    if (!!slots.header) {
        let __VLS_34;
        /** @ts-ignore @type { | typeof __VLS_components.VisuallyHidden | typeof __VLS_components.VisuallyHidden} */
        VisuallyHidden;
        // @ts-ignore
        const __VLS_35 = __VLS_asFunctionalComponent1(__VLS_34, new __VLS_34({}));
        const __VLS_36 = __VLS_35({}, ...__VLS_functionalComponentArgsRest(__VLS_35));
        const { default: __VLS_39 } = __VLS_37.slots;
        let __VLS_40;
        /** @ts-ignore @type { | typeof __VLS_components.DialogTitle} */
        DialogTitle;
        // @ts-ignore
        const __VLS_41 = __VLS_asFunctionalComponent1(__VLS_40, new __VLS_40({}));
        const __VLS_42 = __VLS_41({}, ...__VLS_functionalComponentArgsRest(__VLS_41));
        let __VLS_45;
        /** @ts-ignore @type { | typeof __VLS_components.DialogDescription} */
        DialogDescription;
        // @ts-ignore
        const __VLS_46 = __VLS_asFunctionalComponent1(__VLS_45, new __VLS_45({}));
        const __VLS_47 = __VLS_46({}, ...__VLS_functionalComponentArgsRest(__VLS_46));
        // @ts-ignore
        [isOpen, isOpen, updateOpen, modalId, portalTarget, isMaximize, isMaximize, forceMount, cn, cn, isDragging, isClosed, handleOpenAutoFocus, handleFocusOutside, handleFocusOutside, handleClickOutside, handleClickOutside, handleEscapeKeyDown, handleMaximize, handleAnimationEnd, header,];
        var __VLS_37;
    }
    __VLS_asFunctionalSlot(slots.header)({});
    let __VLS_51;
    /** @ts-ignore @type { | typeof __VLS_components.DialogTitle | typeof __VLS_components.DialogTitle} */
    DialogTitle;
    // @ts-ignore
    const __VLS_52 = __VLS_asFunctionalComponent1(__VLS_51, new __VLS_51({
        ...{ class: "flex-center gap-x-2 sm:justify-start" },
        ...{ class: ({ 'sm:justify-center': props.center }) },
    }));
    const __VLS_53 = __VLS_52({
        ...{ class: "flex-center gap-x-2 sm:justify-start" },
        ...{ class: ({ 'sm:justify-center': props.center }) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_52));
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-x-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['sm:justify-start']} */ ;
    /** @type {__VLS_StyleScopedClasses['sm:justify-center']} */ ;
    const { default: __VLS_56 } = __VLS_54.slots;
    if (props.icon) {
        const __VLS_57 = Icon;
        // @ts-ignore
        const __VLS_58 = __VLS_asFunctionalComponent1(__VLS_57, new __VLS_57({
            name: ({
                info: 'i-ant-design:info-circle-filled',
                success: 'i-ant-design:check-circle-filled',
                warning: 'i-ant-design:exclamation-circle-filled',
                error: 'i-ant-design:close-circle-filled',
            }[props.icon]),
            ...{ class: "size-6" },
            ...{ class: ({
                    'text-blue-600 dark:text-blue-400': props.icon === 'info',
                    'text-green-600 dark:text-green-400': props.icon === 'success',
                    'text-yellow-600 dark:text-yellow-400': props.icon === 'warning',
                    'text-red-600 dark:text-red-400': props.icon === 'error',
                }) },
        }));
        const __VLS_59 = __VLS_58({
            name: ({
                info: 'i-ant-design:info-circle-filled',
                success: 'i-ant-design:check-circle-filled',
                warning: 'i-ant-design:exclamation-circle-filled',
                error: 'i-ant-design:close-circle-filled',
            }[props.icon]),
            ...{ class: "size-6" },
            ...{ class: ({
                    'text-blue-600 dark:text-blue-400': props.icon === 'info',
                    'text-green-600 dark:text-green-400': props.icon === 'success',
                    'text-yellow-600 dark:text-yellow-400': props.icon === 'warning',
                    'text-red-600 dark:text-red-400': props.icon === 'error',
                }) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_58));
        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-blue-600']} */ ;
        /** @type {__VLS_StyleScopedClasses['dark:text-blue-400']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-green-600']} */ ;
        /** @type {__VLS_StyleScopedClasses['dark:text-green-400']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-yellow-600']} */ ;
        /** @type {__VLS_StyleScopedClasses['dark:text-yellow-400']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-red-600']} */ ;
        /** @type {__VLS_StyleScopedClasses['dark:text-red-400']} */ ;
    }
    (typeof __VLS_ctx.title === 'function' ? __VLS_ctx.title() : __VLS_ctx.title);
    // @ts-ignore
    [title, title, title,];
    var __VLS_54;
    let __VLS_62;
    /** @ts-ignore @type { | typeof __VLS_components.DialogDescription | typeof __VLS_components.DialogDescription} */
    DialogDescription;
    // @ts-ignore
    const __VLS_63 = __VLS_asFunctionalComponent1(__VLS_62, new __VLS_62({
        ...{ class: (__VLS_ctx.cn('text-center md:text-start empty:hidden', { 'md:text-center': props.center })) },
    }));
    const __VLS_64 = __VLS_63({
        ...{ class: (__VLS_ctx.cn('text-center md:text-start empty:hidden', { 'md:text-center': props.center })) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_63));
    const { default: __VLS_67 } = __VLS_65.slots;
    (typeof __VLS_ctx.description === 'function' ? __VLS_ctx.description() : __VLS_ctx.description);
    // @ts-ignore
    [cn, description, description, description,];
    var __VLS_65;
    // @ts-ignore
    [];
    var __VLS_29;
}
else {
    let __VLS_68;
    /** @ts-ignore @type { | typeof __VLS_components.VisuallyHidden | typeof __VLS_components.VisuallyHidden} */
    VisuallyHidden;
    // @ts-ignore
    const __VLS_69 = __VLS_asFunctionalComponent1(__VLS_68, new __VLS_68({}));
    const __VLS_70 = __VLS_69({}, ...__VLS_functionalComponentArgsRest(__VLS_69));
    const { default: __VLS_73 } = __VLS_71.slots;
    let __VLS_74;
    /** @ts-ignore @type { | typeof __VLS_components.DialogTitle} */
    DialogTitle;
    // @ts-ignore
    const __VLS_75 = __VLS_asFunctionalComponent1(__VLS_74, new __VLS_74({}));
    const __VLS_76 = __VLS_75({}, ...__VLS_functionalComponentArgsRest(__VLS_75));
    let __VLS_79;
    /** @ts-ignore @type { | typeof __VLS_components.DialogDescription} */
    DialogDescription;
    // @ts-ignore
    const __VLS_80 = __VLS_asFunctionalComponent1(__VLS_79, new __VLS_79({}));
    const __VLS_81 = __VLS_80({}, ...__VLS_functionalComponentArgsRest(__VLS_80));
    // @ts-ignore
    [];
    var __VLS_71;
}
if (!!slots.default) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: (__VLS_ctx.cn('relative flex-1 min-h-40 p-4 overflow-y-auto', props.contentClass)) },
    });
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
const __VLS_85 = Icon;
// @ts-ignore
const __VLS_86 = __VLS_asFunctionalComponent1(__VLS_85, new __VLS_85({
    name: "i-line-md:loading-twotone-loop",
    ...{ class: "size-10" },
}));
const __VLS_87 = __VLS_86({
    name: "i-line-md:loading-twotone-loop",
    ...{ class: "size-10" },
}, ...__VLS_functionalComponentArgsRest(__VLS_86));
/** @type {__VLS_StyleScopedClasses['size-10']} */ ;
if (__VLS_ctx.footer) {
    let __VLS_90;
    /** @ts-ignore @type { | typeof __VLS_components.DialogFooter | typeof __VLS_components.DialogFooter} */
    DialogFooter;
    // @ts-ignore
    const __VLS_91 = __VLS_asFunctionalComponent1(__VLS_90, new __VLS_90({
        ...{ class: (__VLS_ctx.cn('p-3 gap-y-2', props.footerClass, {
                'md:justify-center': props.center,
                'border-t': props.border,
            })) },
    }));
    const __VLS_92 = __VLS_91({
        ...{ class: (__VLS_ctx.cn('p-3 gap-y-2', props.footerClass, {
                'md:justify-center': props.center,
                'border-t': props.border,
            })) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_91));
    const { default: __VLS_95 } = __VLS_93.slots;
    __VLS_asFunctionalSlot(slots.footer)({});
    if (__VLS_ctx.showCancelButton) {
        const __VLS_97 = Button || Button;
        // @ts-ignore
        const __VLS_98 = __VLS_asFunctionalComponent1(__VLS_97, new __VLS_97({
            ...{ 'onClick': {} },
            variant: "outline",
        }));
        const __VLS_99 = __VLS_98({
            ...{ 'onClick': {} },
            variant: "outline",
        }, ...__VLS_functionalComponentArgsRest(__VLS_98));
        let __VLS_102;
        const __VLS_103 = {
            /** @type {typeof __VLS_102.click} */
            onClick: (__VLS_ctx.onCancel),
        };
        const { default: __VLS_104 } = __VLS_100.slots;
        (typeof __VLS_ctx.cancelButtonText === 'function' ? __VLS_ctx.cancelButtonText() : __VLS_ctx.cancelButtonText);
        // @ts-ignore
        [cn, cn, footer, showCancelButton, onCancel, cancelButtonText, cancelButtonText, cancelButtonText,];
        var __VLS_100;
        var __VLS_101;
    }
    if (__VLS_ctx.showConfirmButton) {
        const __VLS_105 = Button || Button;
        // @ts-ignore
        const __VLS_106 = __VLS_asFunctionalComponent1(__VLS_105, new __VLS_105({
            ...{ 'onClick': {} },
            disabled: (__VLS_ctx.confirmButtonDisabled),
            loading: (__VLS_ctx.confirmButtonLoading || __VLS_ctx.isConfirmButtonLoading),
        }));
        const __VLS_107 = __VLS_106({
            ...{ 'onClick': {} },
            disabled: (__VLS_ctx.confirmButtonDisabled),
            loading: (__VLS_ctx.confirmButtonLoading || __VLS_ctx.isConfirmButtonLoading),
        }, ...__VLS_functionalComponentArgsRest(__VLS_106));
        let __VLS_110;
        const __VLS_111 = {
            /** @type {typeof __VLS_110.click} */
            onClick: (__VLS_ctx.onConfirm),
        };
        const { default: __VLS_112 } = __VLS_108.slots;
        (typeof __VLS_ctx.confirmButtonText === 'function' ? __VLS_ctx.confirmButtonText() : __VLS_ctx.confirmButtonText);
        // @ts-ignore
        [showConfirmButton, confirmButtonDisabled, confirmButtonLoading, isConfirmButtonLoading, onConfirm, confirmButtonText, confirmButtonText, confirmButtonText,];
        var __VLS_108;
        var __VLS_109;
    }
    // @ts-ignore
    [];
    var __VLS_93;
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
var __VLS_24 = __VLS_23, __VLS_32 = __VLS_31;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
