import FaButton from '../../button/index.vue';
import { useToast } from '../../toast';
import { useModal } from '../index';
const modal = useModal();
const toast = useToast();
function showInfo() {
    modal.info({
        title: '温馨提醒',
        content: '这是 info 弹窗',
        onConfirm: () => toast('你点了确定'),
    });
}
function showSuccess() {
    modal.success({
        title: '操作成功',
        content: '这是 success 弹窗',
        onConfirm: () => toast('你点了确定'),
    });
}
function showWarning() {
    modal.warning({
        title: '风险提醒',
        content: '这是 warning 弹窗',
        onConfirm: () => toast('你点了确定'),
    });
}
function showError() {
    modal.error({
        title: '操作失败',
        content: '这是 error 弹窗',
        onConfirm: () => toast('你点了确定'),
    });
}
function showConfirm() {
    modal.confirm({
        title: '确认操作',
        content: '这是 confirm 弹窗',
        onConfirm: () => toast('确认操作'),
        onCancel: () => toast('取消操作'),
    });
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-wrap gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
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
    onClick: (__VLS_ctx.showInfo),
};
const { default: __VLS_7 } = __VLS_3.slots;
// @ts-ignore
[showInfo,];
var __VLS_3;
var __VLS_4;
const __VLS_8 = FaButton || FaButton;
// @ts-ignore
const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
    ...{ 'onClick': {} },
    variant: "outline",
}));
const __VLS_10 = __VLS_9({
    ...{ 'onClick': {} },
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
let __VLS_13;
const __VLS_14 = {
    /** @type {typeof __VLS_13.click} */
    onClick: (__VLS_ctx.showSuccess),
};
const { default: __VLS_15 } = __VLS_11.slots;
// @ts-ignore
[showSuccess,];
var __VLS_11;
var __VLS_12;
const __VLS_16 = FaButton || FaButton;
// @ts-ignore
const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
    ...{ 'onClick': {} },
    variant: "outline",
}));
const __VLS_18 = __VLS_17({
    ...{ 'onClick': {} },
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_17));
let __VLS_21;
const __VLS_22 = {
    /** @type {typeof __VLS_21.click} */
    onClick: (__VLS_ctx.showWarning),
};
const { default: __VLS_23 } = __VLS_19.slots;
// @ts-ignore
[showWarning,];
var __VLS_19;
var __VLS_20;
const __VLS_24 = FaButton || FaButton;
// @ts-ignore
const __VLS_25 = __VLS_asFunctionalComponent1(__VLS_24, new __VLS_24({
    ...{ 'onClick': {} },
    variant: "outline",
}));
const __VLS_26 = __VLS_25({
    ...{ 'onClick': {} },
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_25));
let __VLS_29;
const __VLS_30 = {
    /** @type {typeof __VLS_29.click} */
    onClick: (__VLS_ctx.showError),
};
const { default: __VLS_31 } = __VLS_27.slots;
// @ts-ignore
[showError,];
var __VLS_27;
var __VLS_28;
const __VLS_32 = FaButton || FaButton;
// @ts-ignore
const __VLS_33 = __VLS_asFunctionalComponent1(__VLS_32, new __VLS_32({
    ...{ 'onClick': {} },
    variant: "secondary",
}));
const __VLS_34 = __VLS_33({
    ...{ 'onClick': {} },
    variant: "secondary",
}, ...__VLS_functionalComponentArgsRest(__VLS_33));
let __VLS_37;
const __VLS_38 = {
    /** @type {typeof __VLS_37.click} */
    onClick: (__VLS_ctx.showConfirm),
};
const { default: __VLS_39 } = __VLS_35.slots;
// @ts-ignore
[showConfirm,];
var __VLS_35;
var __VLS_36;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
