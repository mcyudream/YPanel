import FaButton from '../../button/index.vue';
import { useImagePreview } from '../index';
const { open } = useImagePreview();
function openSingle() {
    open('https://fantastic-admin.hurui.me/logo.svg');
}
function openMulti() {
    open([
        'https://fantastic-admin.hurui.me/logo.svg',
        'https://fantastic-mobile.hurui.me/logo.png',
    ]);
}
function openMultiWithIndex() {
    open([
        'https://fantastic-admin.hurui.me/logo.svg',
        'https://fantastic-mobile.hurui.me/logo.png',
    ], 1);
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
    onClick: (__VLS_ctx.openSingle),
};
const { default: __VLS_7 } = __VLS_3.slots;
// @ts-ignore
[openSingle,];
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
    onClick: (__VLS_ctx.openMulti),
};
const { default: __VLS_15 } = __VLS_11.slots;
// @ts-ignore
[openMulti,];
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
    onClick: (__VLS_ctx.openMultiWithIndex),
};
const { default: __VLS_23 } = __VLS_19.slots;
// @ts-ignore
[openMultiWithIndex,];
var __VLS_19;
var __VLS_20;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
