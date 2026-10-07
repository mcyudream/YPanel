import { shallowRef } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../../button/index.vue';
import FaDrawer from '../index.vue';
const centeredOpen = shallowRef(false);
const borderlessOpen = shallowRef(false);
const loadingOpen = shallowRef(false);
const loading = shallowRef(false);
function openLoadingDrawer() {
    loadingOpen.value = true;
    loading.value = true;
    window.setTimeout(() => {
        loading.value = false;
    }, 1800);
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
    onClick: (...[$event]) => {
        return (__VLS_ctx.centeredOpen = true);
        // @ts-ignore
        [centeredOpen,];
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
        return (__VLS_ctx.borderlessOpen = true);
        // @ts-ignore
        [borderlessOpen,];
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
    variant: "secondary",
}));
const __VLS_18 = __VLS_17({
    ...{ 'onClick': {} },
    variant: "secondary",
}, ...__VLS_functionalComponentArgsRest(__VLS_17));
let __VLS_21;
const __VLS_22 = {
    /** @type {typeof __VLS_21.click} */
    onClick: (__VLS_ctx.openLoadingDrawer),
};
const { default: __VLS_23 } = __VLS_19.slots;
// @ts-ignore
[openLoadingDrawer,];
var __VLS_19;
var __VLS_20;
const __VLS_24 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_25 = __VLS_asFunctionalComponent1(__VLS_24, new __VLS_24({
    modelValue: (__VLS_ctx.centeredOpen),
    title: "居中抽屉",
    centered: true,
    showCancelButton: true,
}));
const __VLS_26 = __VLS_25({
    modelValue: (__VLS_ctx.centeredOpen),
    title: "居中抽屉",
    centered: true,
    showCancelButton: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_25));
const { default: __VLS_29 } = __VLS_27.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground text-center" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
/** @type {__VLS_StyleScopedClasses['text-center']} */ ;
// @ts-ignore
[centeredOpen,];
var __VLS_27;
const __VLS_30 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({
    modelValue: (__VLS_ctx.borderlessOpen),
    title: "无边框抽屉",
    bordered: (false),
    showCancelButton: true,
}));
const __VLS_32 = __VLS_31({
    modelValue: (__VLS_ctx.borderlessOpen),
    title: "无边框抽屉",
    bordered: (false),
    showCancelButton: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_31));
const { default: __VLS_35 } = __VLS_33.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
// @ts-ignore
[borderlessOpen,];
var __VLS_33;
const __VLS_36 = FaDrawer || FaDrawer;
// @ts-ignore
const __VLS_37 = __VLS_asFunctionalComponent1(__VLS_36, new __VLS_36({
    modelValue: (__VLS_ctx.loadingOpen),
    title: "载入状态",
    loading: (__VLS_ctx.loading),
}));
const __VLS_38 = __VLS_37({
    modelValue: (__VLS_ctx.loadingOpen),
    title: "载入状态",
    loading: (__VLS_ctx.loading),
}, ...__VLS_functionalComponentArgsRest(__VLS_37));
const { default: __VLS_41 } = __VLS_39.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
// @ts-ignore
[loadingOpen, loading,];
var __VLS_39;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
