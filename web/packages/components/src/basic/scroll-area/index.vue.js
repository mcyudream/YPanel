import { useElementSize, useTextDirection } from '@vueuse/core';
import { onMounted, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import { ScrollArea as ScrollAreaRoot, ScrollBar } from './scroll-area';
defineOptions({
    name: 'BuiltInScrollArea',
});
const props = withDefaults(defineProps(), {
    horizontal: false,
    scrollbar: true,
    mask: false,
});
const emits = defineEmits();
const dir = useTextDirection({
    observe: true,
});
const scrollAreaRef = useTemplateRef('scrollAreaRef');
function onScroll(event) {
    emits('onScroll', event);
}
function onWheel(event) {
    if (props.horizontal) {
        scrollAreaRef.value?.el?.viewportElement?.scrollBy({
            left: event.deltaY || event.detail,
        });
    }
}
const scrollContainerRef = useTemplateRef({});
onMounted(() => {
    const { width, height } = useElementSize(scrollContainerRef.value);
    watch([width, height], () => {
        scrollAreaRef.value?.el?.viewportElement?.dispatchEvent(new Event('scroll'));
    }, {
        immediate: true,
    });
});
function scrollTo(scrollNumber, behavior = 'auto') {
    if (props.horizontal) {
        scrollAreaRef.value?.el?.viewportElement?.scrollTo({
            left: scrollNumber,
            behavior,
        });
    }
    else {
        scrollAreaRef.value?.el?.viewportElement?.scrollTo({
            top: scrollNumber,
            behavior,
        });
    }
}
let __VLS_exposed;
defineExpose({
    ref: scrollAreaRef,
    scrollTo,
});
const __VLS_defaults = {
    horizontal: false,
    scrollbar: true,
    mask: false,
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ref: "scrollContainerRef",
    ...{ class: (__VLS_ctx.cn('relative flex overflow-hidden', props.class)) },
});
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.ScrollAreaRoot | typeof __VLS_components.ScrollAreaRoot} */
ScrollAreaRoot;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ref: "scrollAreaRef",
    ...{ class: (__VLS_ctx.cn('relative z-0 flex-1', props.contentClass)) },
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
    horizontal: (props.horizontal),
    scrollbar: (props.scrollbar),
    mask: (props.mask),
    onScroll: (__VLS_ctx.onScroll),
    onWheel: (__VLS_ctx.onWheel),
}));
const __VLS_2 = __VLS_1({
    ref: "scrollAreaRef",
    ...{ class: (__VLS_ctx.cn('relative z-0 flex-1', props.contentClass)) },
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
    horizontal: (props.horizontal),
    scrollbar: (props.scrollbar),
    mask: (props.mask),
    onScroll: (__VLS_ctx.onScroll),
    onWheel: (__VLS_ctx.onWheel),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_7 } = __VLS_3.slots;
var __VLS_8 = {};
if (props.horizontal) {
    let __VLS_10;
    /** @ts-ignore @type { | typeof __VLS_components.ScrollBar} */
    ScrollBar;
    // @ts-ignore
    const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
        orientation: "horizontal",
        ...{ class: ({ 'opacity-0 pointer-events-none': !props.scrollbar }) },
    }));
    const __VLS_12 = __VLS_11({
        orientation: "horizontal",
        ...{ class: ({ 'opacity-0 pointer-events-none': !props.scrollbar }) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_11));
    /** @type {__VLS_StyleScopedClasses['opacity-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['pointer-events-none']} */ ;
}
// @ts-ignore
[cn, cn, dir, onScroll, onWheel,];
var __VLS_3;
// @ts-ignore
var __VLS_6 = __VLS_5, __VLS_9 = __VLS_8;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    setup: () => __VLS_exposed,
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
