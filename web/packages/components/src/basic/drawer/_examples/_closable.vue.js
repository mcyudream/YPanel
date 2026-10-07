import { shallowRef } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../../button/index.vue';
import FaDrawer from '../index.vue';
const closableOpen = shallowRef(false);
const notClosableOpen = shallowRef(false);
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
    onClick: (...[$event]) => {
        return (__VLS_ctx.closableOpen = true);
        // @ts-ignore
        [closableOpen,];
    },
};
const { default: __VLS_7 } = __VLS_3.slots;
// @ts-ignore
[];
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
    onClick: (...[$event]) => {
        return (__VLS_ctx.notClosableOpen = true);
        // @ts-ignore
        [notClosableOpen,];
    },
};
const { default: __VLS_15 } = __VLS_11.slots;
// @ts-ignore
[];
var __VLS_11;
var __VLS_12;
const __VLS_16 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
    modelValue: (__VLS_ctx.closableOpen),
    title: "显示关闭按钮",
    closable: true,
}));
const __VLS_18 = __VLS_17({
    modelValue: (__VLS_ctx.closableOpen),
    title: "显示关闭按钮",
    closable: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_17));
const { default: __VLS_21 } = __VLS_19.slots;
// @ts-ignore
[closableOpen,];
var __VLS_19;
const __VLS_22 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_23 = __VLS_asFunctionalComponent1(__VLS_22, new __VLS_22({
    modelValue: (__VLS_ctx.notClosableOpen),
    title: "隐藏关闭按钮",
    closable: (false),
}));
const __VLS_24 = __VLS_23({
    modelValue: (__VLS_ctx.notClosableOpen),
    title: "隐藏关闭按钮",
    closable: (false),
}, ...__VLS_functionalComponentArgsRest(__VLS_23));
const { default: __VLS_27 } = __VLS_25.slots;
// @ts-ignore
[notClosableOpen,];
var __VLS_25;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
