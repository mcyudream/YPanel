import { shallowRef } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../../button/index.vue';
import { useToast } from '../../toast';
import FaModal from '../index.vue';
const open = shallowRef(false);
const toast = useToast();
function handleBeforeClose(action, done) {
    toast(`beforeClose: ${action}`);
    window.setTimeout(done, 600);
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaButton || FaButton;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onClick': {} },
}));
const __VLS_2 = __VLS_1({
    ...{ 'onClick': {} },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.click} */
    onClick: (...[$event]) => {
        return (__VLS_ctx.open = true);
        // @ts-ignore
        [open,];
    },
};
const { default: __VLS_7 } = __VLS_3.slots;
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
const __VLS_8 = FaModal || FaModal;
// @ts-ignore
const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
    ...{ 'onOpen': {} },
    ...{ 'onOpened': {} },
    ...{ 'onClose': {} },
    ...{ 'onClosed': {} },
    ...{ 'onConfirm': {} },
    ...{ 'onCancel': {} },
    modelValue: (__VLS_ctx.open),
    title: "触发事件",
    showCancelButton: true,
    beforeClose: (__VLS_ctx.handleBeforeClose),
}));
const __VLS_10 = __VLS_9({
    ...{ 'onOpen': {} },
    ...{ 'onOpened': {} },
    ...{ 'onClose': {} },
    ...{ 'onClosed': {} },
    ...{ 'onConfirm': {} },
    ...{ 'onCancel': {} },
    modelValue: (__VLS_ctx.open),
    title: "触发事件",
    showCancelButton: true,
    beforeClose: (__VLS_ctx.handleBeforeClose),
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
let __VLS_13;
const __VLS_14 = {
    /** @type {typeof __VLS_13.open} */
    onOpen: (...[$event]) => {
        return (__VLS_ctx.toast('open'));
        // @ts-ignore
        [open, handleBeforeClose, toast,];
    },
};
const __VLS_15 = {
    /** @type {typeof __VLS_13.opened} */
    onOpened: (...[$event]) => {
        return (__VLS_ctx.toast('opened'));
        // @ts-ignore
        [toast,];
    },
};
const __VLS_16 = {
    /** @type {typeof __VLS_13.close} */
    onClose: (...[$event]) => {
        return (__VLS_ctx.toast('close'));
        // @ts-ignore
        [toast,];
    },
};
const __VLS_17 = {
    /** @type {typeof __VLS_13.closed} */
    onClosed: (...[$event]) => {
        return (__VLS_ctx.toast('closed'));
        // @ts-ignore
        [toast,];
    },
};
const __VLS_18 = {
    /** @type {typeof __VLS_13.confirm} */
    onConfirm: (...[$event]) => {
        return (__VLS_ctx.toast('confirm'));
        // @ts-ignore
        [toast,];
    },
};
const __VLS_19 = {
    /** @type {typeof __VLS_13.cancel} */
    onCancel: (...[$event]) => {
        return (__VLS_ctx.toast('cancel'));
        // @ts-ignore
        [toast,];
    },
};
const { default: __VLS_20 } = __VLS_11.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
// @ts-ignore
[];
var __VLS_11;
var __VLS_12;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
