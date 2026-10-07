import { reactiveOmit } from '@vueuse/core';
import { ScrollAreaScrollbar, ScrollAreaThumb } from 'reka-ui';
import { cn } from '#utils';
const props = withDefaults(defineProps(), {
    orientation: 'vertical',
});
const delegatedProps = reactiveOmit(props, 'class');
const __VLS_defaults = {
    orientation: 'vertical',
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.ScrollAreaScrollbar | typeof __VLS_components.ScrollAreaScrollbar} */
ScrollAreaScrollbar;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "scroll-area-scrollbar",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('flex touch-none p-px transition-colors select-none', __VLS_ctx.orientation === 'vertical'
            && 'h-full w-2.5 border-l border-l-transparent', __VLS_ctx.orientation === 'horizontal'
            && 'h-2.5 flex-col border-t border-t-transparent', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "scroll-area-scrollbar",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('flex touch-none p-px transition-colors select-none', __VLS_ctx.orientation === 'vertical'
            && 'h-full w-2.5 border-l border-l-transparent', __VLS_ctx.orientation === 'horizontal'
            && 'h-2.5 flex-col border-t border-t-transparent', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.ScrollAreaThumb} */
ScrollAreaThumb;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    dataSlot: "scroll-area-thumb",
    ...{ class: "rounded-full bg-border flex-1 relative" },
}));
const __VLS_9 = __VLS_8({
    dataSlot: "scroll-area-thumb",
    ...{ class: "rounded-full bg-border flex-1 relative" },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
/** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-border']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
/** @type {__VLS_StyleScopedClasses['relative']} */ ;
// @ts-ignore
[delegatedProps, cn, orientation, orientation,];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
export default {};
