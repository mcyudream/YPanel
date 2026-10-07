import { reactiveOmit, useResizeObserver } from '@vueuse/core';
import { ScrollAreaCorner, ScrollAreaRoot, ScrollAreaViewport, } from 'reka-ui';
import { computed, ref, useTemplateRef } from 'vue';
import { cn } from '#utils';
import ScrollBar from './ScrollBar.vue';
const props = defineProps();
const delegatedProps = reactiveOmit(props, 'class');
const viewportRef = useTemplateRef({});
const canScroll = ref(false);
function checkCanScroll() {
    const el = viewportRef.value?.$el ?? viewportRef.value;
    if (!el) {
        return;
    }
    canScroll.value = props.horizontal
        ? el.scrollWidth > el.clientWidth
        : el.scrollHeight > el.clientHeight;
}
useResizeObserver(computed(() => {
    const el = viewportRef.value?.$el ?? viewportRef.value;
    // 同时观察 viewport 和其内容子元素，以便内容尺寸变化时也能触发
    return el ? [el, el.firstElementChild].filter(Boolean) : [];
}), checkCanScroll);
const __VLS_exposed = {
    el: viewportRef,
};
defineExpose(__VLS_exposed);
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
/** @type {__VLS_StyleScopedClasses['scroll-area-mask-vertical']} */ ;
/** @type {__VLS_StyleScopedClasses['can-scroll']} */ ;
/** @type {__VLS_StyleScopedClasses['scroll-area-mask-horizontal']} */ ;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.ScrollAreaRoot | typeof __VLS_components.ScrollAreaRoot} */
ScrollAreaRoot;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "scroll-area",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('relative overflow-hidden', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "scroll-area",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('relative overflow-hidden', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.ScrollAreaViewport | typeof __VLS_components.ScrollAreaViewport} */
ScrollAreaViewport;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    ...{ 'onScroll': {} },
    ...{ 'onWheel': {} },
    ref: "viewportRef",
    dataSlot: "scroll-area-viewport",
    ...{ class: "scroll-area-viewport outline-none rounded-[inherit] size-full transition-[color,box-shadow] focus-visible:outline-1 focus-visible:ring-3 focus-visible:ring-ring/50" },
    ...{ class: ({
            'scroll-area-mask-vertical': props.mask && !props.horizontal,
            'scroll-area-mask-horizontal': props.mask && props.horizontal,
            'can-scroll': __VLS_ctx.canScroll,
        }) },
    ...{ style: ({ scrollTimelineName: '--scroll-area-mask-timeline', scrollTimelineAxis: props.horizontal ? 'x' : 'y' }) },
}));
const __VLS_9 = __VLS_8({
    ...{ 'onScroll': {} },
    ...{ 'onWheel': {} },
    ref: "viewportRef",
    dataSlot: "scroll-area-viewport",
    ...{ class: "scroll-area-viewport outline-none rounded-[inherit] size-full transition-[color,box-shadow] focus-visible:outline-1 focus-visible:ring-3 focus-visible:ring-ring/50" },
    ...{ class: ({
            'scroll-area-mask-vertical': props.mask && !props.horizontal,
            'scroll-area-mask-horizontal': props.mask && props.horizontal,
            'can-scroll': __VLS_ctx.canScroll,
        }) },
    ...{ style: ({ scrollTimelineName: '--scroll-area-mask-timeline', scrollTimelineAxis: props.horizontal ? 'x' : 'y' }) },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
let __VLS_12;
const __VLS_13 = {
    /** @type {typeof __VLS_12.scroll} */
    onScroll: (__VLS_ctx.onScroll),
};
const __VLS_14 = {
    /** @type {typeof __VLS_12.wheel} */
    onWheel: (__VLS_ctx.onWheel),
};
var __VLS_15;
/** @type {__VLS_StyleScopedClasses['scroll-area-viewport']} */ ;
/** @type {__VLS_StyleScopedClasses['outline-none']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-[inherit]']} */ ;
/** @type {__VLS_StyleScopedClasses['size-full']} */ ;
/** @type {__VLS_StyleScopedClasses['transition-[color,box-shadow]']} */ ;
/** @type {__VLS_StyleScopedClasses['focus-visible:outline-1']} */ ;
/** @type {__VLS_StyleScopedClasses['focus-visible:ring-3']} */ ;
/** @type {__VLS_StyleScopedClasses['focus-visible:ring-ring/50']} */ ;
/** @type {__VLS_StyleScopedClasses['scroll-area-mask-vertical']} */ ;
/** @type {__VLS_StyleScopedClasses['scroll-area-mask-horizontal']} */ ;
/** @type {__VLS_StyleScopedClasses['can-scroll']} */ ;
const { default: __VLS_17 } = __VLS_10.slots;
var __VLS_18 = {};
// @ts-ignore
[delegatedProps, cn, canScroll, onScroll, onWheel,];
var __VLS_10;
var __VLS_11;
const __VLS_20 = ScrollBar;
// @ts-ignore
const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({
    ...{ class: ({ 'opacity-0 pointer-events-none': !props.scrollbar }) },
}));
const __VLS_22 = __VLS_21({
    ...{ class: ({ 'opacity-0 pointer-events-none': !props.scrollbar }) },
}, ...__VLS_functionalComponentArgsRest(__VLS_21));
/** @type {__VLS_StyleScopedClasses['opacity-0']} */ ;
/** @type {__VLS_StyleScopedClasses['pointer-events-none']} */ ;
let __VLS_25;
/** @ts-ignore @type { | typeof __VLS_components.ScrollAreaCorner} */
ScrollAreaCorner;
// @ts-ignore
const __VLS_26 = __VLS_asFunctionalComponent1(__VLS_25, new __VLS_25({}));
const __VLS_27 = __VLS_26({}, ...__VLS_functionalComponentArgsRest(__VLS_26));
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
var __VLS_16 = __VLS_15, __VLS_19 = __VLS_18;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    setup: () => __VLS_exposed,
    __typeProps: {},
});
const __VLS_export = {};
export default {};
