import FaButton from '../../button/index.vue';
import { useToast } from '../index';
const toast = useToast();
function showLoading() {
    const loading = toast.loading('正在处理...', {
        duration: Infinity,
    });
    setTimeout(() => {
        toast.dismiss(loading);
        toast.success('处理完成');
    }, 2000);
}
function showPromise() {
    toast.promise(() => new Promise((resolve) => {
        setTimeout(resolve, 2000);
    }), {
        loading: '正在加载数据',
        success: () => '数据加载完成',
        error: () => '数据加载失败',
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
    ...{ class: "flex flex-wrap gap-4" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
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
    onClick: (__VLS_ctx.showLoading),
};
const { default: __VLS_7 } = __VLS_3.slots;
// @ts-ignore
[showLoading,];
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
    onClick: (__VLS_ctx.showPromise),
};
const { default: __VLS_15 } = __VLS_11.slots;
// @ts-ignore
[showPromise,];
var __VLS_11;
var __VLS_12;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
