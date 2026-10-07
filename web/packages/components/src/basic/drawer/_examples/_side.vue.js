import { reactive } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../../button/index.vue';
import FaDrawer from '../index.vue';
const drawers = reactive({
    top: false,
    bottom: false,
    left: false,
    right: false,
});
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
        return (__VLS_ctx.drawers.top = true);
        // @ts-ignore
        [drawers,];
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
}));
const __VLS_10 = __VLS_9({
    ...{ 'onClick': {} },
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
let __VLS_13;
const __VLS_14 = {
    /** @type {typeof __VLS_13.click} */
    onClick: (...[$event]) => {
        return (__VLS_ctx.drawers.bottom = true);
        // @ts-ignore
        [drawers,];
    },
};
const { default: __VLS_15 } = __VLS_11.slots;
// @ts-ignore
[];
var __VLS_11;
var __VLS_12;
const __VLS_16 = FaButton || FaButton;
// @ts-ignore
const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
    ...{ 'onClick': {} },
}));
const __VLS_18 = __VLS_17({
    ...{ 'onClick': {} },
}, ...__VLS_functionalComponentArgsRest(__VLS_17));
let __VLS_21;
const __VLS_22 = {
    /** @type {typeof __VLS_21.click} */
    onClick: (...[$event]) => {
        return (__VLS_ctx.drawers.left = true);
        // @ts-ignore
        [drawers,];
    },
};
const { default: __VLS_23 } = __VLS_19.slots;
// @ts-ignore
[];
var __VLS_19;
var __VLS_20;
const __VLS_24 = FaButton || FaButton;
// @ts-ignore
const __VLS_25 = __VLS_asFunctionalComponent1(__VLS_24, new __VLS_24({
    ...{ 'onClick': {} },
}));
const __VLS_26 = __VLS_25({
    ...{ 'onClick': {} },
}, ...__VLS_functionalComponentArgsRest(__VLS_25));
let __VLS_29;
const __VLS_30 = {
    /** @type {typeof __VLS_29.click} */
    onClick: (...[$event]) => {
        return (__VLS_ctx.drawers.right = true);
        // @ts-ignore
        [drawers,];
    },
};
const { default: __VLS_31 } = __VLS_27.slots;
// @ts-ignore
[];
var __VLS_27;
var __VLS_28;
const __VLS_32 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_33 = __VLS_asFunctionalComponent1(__VLS_32, new __VLS_32({
    modelValue: (__VLS_ctx.drawers.top),
    side: "top",
    title: "上方抽屉",
}));
const __VLS_34 = __VLS_33({
    modelValue: (__VLS_ctx.drawers.top),
    side: "top",
    title: "上方抽屉",
}, ...__VLS_functionalComponentArgsRest(__VLS_33));
const { default: __VLS_37 } = __VLS_35.slots;
// @ts-ignore
[drawers,];
var __VLS_35;
const __VLS_38 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({
    modelValue: (__VLS_ctx.drawers.bottom),
    side: "bottom",
    title: "下方抽屉",
}));
const __VLS_40 = __VLS_39({
    modelValue: (__VLS_ctx.drawers.bottom),
    side: "bottom",
    title: "下方抽屉",
}, ...__VLS_functionalComponentArgsRest(__VLS_39));
const { default: __VLS_43 } = __VLS_41.slots;
// @ts-ignore
[drawers,];
var __VLS_41;
const __VLS_44 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
    modelValue: (__VLS_ctx.drawers.left),
    side: "left",
    title: "左侧抽屉",
}));
const __VLS_46 = __VLS_45({
    modelValue: (__VLS_ctx.drawers.left),
    side: "left",
    title: "左侧抽屉",
}, ...__VLS_functionalComponentArgsRest(__VLS_45));
const { default: __VLS_49 } = __VLS_47.slots;
// @ts-ignore
[drawers,];
var __VLS_47;
const __VLS_50 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_51 = __VLS_asFunctionalComponent1(__VLS_50, new __VLS_50({
    modelValue: (__VLS_ctx.drawers.right),
    side: "right",
    title: "右侧抽屉",
}));
const __VLS_52 = __VLS_51({
    modelValue: (__VLS_ctx.drawers.right),
    side: "right",
    title: "右侧抽屉",
}, ...__VLS_functionalComponentArgsRest(__VLS_51));
const { default: __VLS_55 } = __VLS_53.slots;
// @ts-ignore
[drawers,];
var __VLS_53;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
