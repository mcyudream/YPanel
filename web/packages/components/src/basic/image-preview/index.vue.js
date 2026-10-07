import { ref, watch } from 'vue';
import { cn } from '#utils';
import Icon from '../icon/index.vue';
import Preview from './preview.vue';
defineOptions({
    name: 'BuiltInImagePreview',
});
const props = defineProps();
const emits = defineEmits();
const isLoading = ref(true);
const isError = ref(false);
const isOpen = ref(false);
// 监听 src 变化，重置加载状态
watch(() => props.src, () => {
    isLoading.value = true;
    isError.value = false;
}, { immediate: false });
function handleLoad() {
    isLoading.value = false;
    emits('load');
}
function handleError() {
    isError.value = true;
    isLoading.value = false;
    emits('error');
}
function handleClick() {
    if (!isError.value) {
        isOpen.value = true;
    }
}
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
    ...{ class: "group/image-preview border rounded-lg inline-block relative overflow-hidden" },
});
/** @type {__VLS_StyleScopedClasses['group/image-preview']} */ ;
/** @type {__VLS_StyleScopedClasses['border']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
/** @type {__VLS_StyleScopedClasses['inline-block']} */ ;
/** @type {__VLS_StyleScopedClasses['relative']} */ ;
/** @type {__VLS_StyleScopedClasses['overflow-hidden']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.img)({
    ...{ onError: (__VLS_ctx.handleError) },
    ...{ onLoad: (__VLS_ctx.handleLoad) },
    ...{ onClick: (...[$event]) => {
            return (!__VLS_ctx.isLoading && !__VLS_ctx.isError && __VLS_ctx.handleClick());
            // @ts-ignore
            [handleError, handleLoad, isLoading, isError, handleClick,];
        } },
    src: (__VLS_ctx.src),
    ...{ class: (__VLS_ctx.cn('size-50 object-contain cursor-pointer transition-all duration-300 group-hover/image-preview:scale-110', props.class, {
            invisible: __VLS_ctx.isError || __VLS_ctx.isLoading,
        })) },
});
if (__VLS_ctx.isLoading) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex-center h-full w-full left-0 top-0 absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['h-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['left-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    var __VLS_0 = {};
    const __VLS_2 = Icon;
    // @ts-ignore
    const __VLS_3 = __VLS_asFunctionalComponent1(__VLS_2, new __VLS_2({
        name: "i-line-md:loading-twotone-loop",
        ...{ class: "text-secondary-foreground/50 size-8" },
    }));
    const __VLS_4 = __VLS_3({
        name: "i-line-md:loading-twotone-loop",
        ...{ class: "text-secondary-foreground/50 size-8" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_3));
    /** @type {__VLS_StyleScopedClasses['text-secondary-foreground/50']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-8']} */ ;
}
if (__VLS_ctx.isError) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex-center h-full w-full left-0 top-0 absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['h-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['left-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    var __VLS_7 = {};
    const __VLS_9 = Icon;
    // @ts-ignore
    const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
        name: "i-ph:image-broken-duotone",
        ...{ class: "text-secondary-foreground/50 size-8" },
    }));
    const __VLS_11 = __VLS_10({
        name: "i-ph:image-broken-duotone",
        ...{ class: "text-secondary-foreground/50 size-8" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_10));
    /** @type {__VLS_StyleScopedClasses['text-secondary-foreground/50']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-8']} */ ;
}
const __VLS_14 = Preview;
// @ts-ignore
const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
    modelValue: (__VLS_ctx.isOpen),
    src: __VLS_ctx.src,
}));
const __VLS_16 = __VLS_15({
    modelValue: (__VLS_ctx.isOpen),
    src: __VLS_ctx.src,
}, ...__VLS_functionalComponentArgsRest(__VLS_15));
// @ts-ignore
var __VLS_1 = __VLS_0, __VLS_8 = __VLS_7;
// @ts-ignore
[isLoading, isLoading, isError, isError, src, src, cn, isOpen,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
