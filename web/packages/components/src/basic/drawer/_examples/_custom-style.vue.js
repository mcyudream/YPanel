import { shallowRef } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../../button/index.vue';
import FaDrawer from '../index.vue';
const open = shallowRef(false);
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
const __VLS_8 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
    modelValue: (__VLS_ctx.open),
    title: "自定义样式",
    description: "通过 contentClass、headerClass 和 footerClass 调整抽屉区域样式。",
    contentClass: "sm:max-w-xl border-primary/30",
    headerClass: "bg-primary/8",
    footerClass: "bg-muted/50",
    showCancelButton: true,
}));
const __VLS_10 = __VLS_9({
    modelValue: (__VLS_ctx.open),
    title: "自定义样式",
    description: "通过 contentClass、headerClass 和 footerClass 调整抽屉区域样式。",
    contentClass: "sm:max-w-xl border-primary/30",
    headerClass: "bg-primary/8",
    footerClass: "bg-muted/50",
    showCancelButton: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
const { default: __VLS_13 } = __VLS_11.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm leading-6 p-4 rounded-lg bg-muted/50" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['leading-6']} */ ;
/** @type {__VLS_StyleScopedClasses['p-4']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
// @ts-ignore
[open,];
var __VLS_11;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
