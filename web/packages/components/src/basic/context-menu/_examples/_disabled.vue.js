import FaIcon from '../../icon/index.vue';
import { useToast } from '../../toast';
import FaContextMenu from '../index.vue';
const toast = useToast();
function handleClick(text) {
    toast(text);
}
const items = [
    [
        { label: '打开', handle: () => handleClick('打开') },
        { label: '复制', disabled: true, handle: () => handleClick('复制') },
        { label: '移动到', disabled: true, handle: () => handleClick('移动到') },
    ],
    [
        { label: '刷新', handle: () => handleClick('刷新') },
    ],
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaContextMenu || FaContextMenu;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    items: (__VLS_ctx.items),
}));
const __VLS_2 = __VLS_1({
    items: (__VLS_ctx.items),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm border rounded-md border-dashed flex h-[150px] w-[300px] items-center justify-center" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['border']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
/** @type {__VLS_StyleScopedClasses['border-dashed']} */ ;
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['h-[150px]']} */ ;
/** @type {__VLS_StyleScopedClasses['w-[300px]']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
/** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
const __VLS_7 = FaIcon;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    name: "i-hugeicons:mouse-right-click-06",
    ...{ class: "op-50 size-12" },
}));
const __VLS_9 = __VLS_8({
    name: "i-hugeicons:mouse-right-click-06",
    ...{ class: "op-50 size-12" },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
/** @type {__VLS_StyleScopedClasses['op-50']} */ ;
/** @type {__VLS_StyleScopedClasses['size-12']} */ ;
// @ts-ignore
[items,];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
