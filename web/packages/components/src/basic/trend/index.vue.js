import { computed } from 'vue';
import Icon from '../icon/index.vue';
defineOptions({
    name: 'BuiltInTrend',
});
const props = withDefaults(defineProps(), {
    type: 'up',
    prefix: '',
    suffix: '',
    reverse: false,
    size: 'medium',
    variant: 'default',
});
const isUp = computed(() => props.type === 'up');
const isColorUp = computed(() => {
    return props.reverse ? !isUp.value : isUp.value;
});
const sizeClasses = {
    small: 'text-xs px-1.5 py-0.5 gap-0.5',
    medium: 'text-sm px-2 py-1 gap-1',
    large: 'text-base px-2.5 py-1.5 gap-1.5',
};
const __VLS_defaults = {
    type: 'up',
    prefix: '',
    suffix: '',
    reverse: false,
    size: 'medium',
    variant: 'default',
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
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "font-medium rounded-full inline-flex transition-all duration-300 items-center" },
    ...{ class: ([
            __VLS_ctx.sizeClasses[__VLS_ctx.size],
            __VLS_ctx.variant === 'default' && [__VLS_ctx.isColorUp ? 'text-green-500' : 'text-red-500'],
            __VLS_ctx.variant === 'filled' && [
                __VLS_ctx.isColorUp ? 'bg-gradient-to-r from-green-500 to-emerald-500 text-white shadow-lg shadow-green-500/30' : 'bg-gradient-to-r from-red-500 to-rose-500 text-white shadow-lg shadow-red-500/30',
            ],
            __VLS_ctx.variant === 'soft' && [
                __VLS_ctx.isColorUp ? 'bg-green-500/10 text-green-600 dark:bg-green-500/20 dark:text-green-400' : 'bg-red-500/10 text-red-600 dark:bg-red-500/20 dark:text-red-400',
            ],
            __VLS_ctx.variant === 'outline' && [
                __VLS_ctx.isColorUp ? 'border border-green-500/30 text-green-600 dark:text-green-400 bg-green-500/5' : 'border border-red-500/30 text-red-600 dark:text-red-400 bg-red-500/5',
            ],
        ]) },
});
/** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
/** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
/** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
/** @type {__VLS_StyleScopedClasses['duration-300']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
if (__VLS_ctx.prefix) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "opacity-70" },
    });
    /** @type {__VLS_StyleScopedClasses['opacity-70']} */ ;
    (__VLS_ctx.prefix);
}
__VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
    ...{ class: "tabular-nums" },
});
/** @type {__VLS_StyleScopedClasses['tabular-nums']} */ ;
(__VLS_ctx.value);
if (__VLS_ctx.suffix) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "opacity-70" },
    });
    /** @type {__VLS_StyleScopedClasses['opacity-70']} */ ;
    (__VLS_ctx.suffix);
}
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex items-center justify-center relative" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
/** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
/** @type {__VLS_StyleScopedClasses['relative']} */ ;
const __VLS_0 = Icon;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    name: (__VLS_ctx.isUp ? 'i-ep:caret-top' : 'i-ep:caret-top'),
    rotate: (__VLS_ctx.isUp ? 0 : 180),
    ...{ class: "transition-transform duration-300" },
    ...{ class: ([
            __VLS_ctx.size === 'small' ? 'text-[10px]' : __VLS_ctx.size === 'large' ? 'text-lg' : 'text-sm',
            __VLS_ctx.variant === 'filled' ? 'text-white' : (__VLS_ctx.isColorUp ? 'text-green-500' : 'text-red-500'),
        ]) },
}));
const __VLS_2 = __VLS_1({
    name: (__VLS_ctx.isUp ? 'i-ep:caret-top' : 'i-ep:caret-top'),
    rotate: (__VLS_ctx.isUp ? 0 : 180),
    ...{ class: "transition-transform duration-300" },
    ...{ class: ([
            __VLS_ctx.size === 'small' ? 'text-[10px]' : __VLS_ctx.size === 'large' ? 'text-lg' : 'text-sm',
            __VLS_ctx.variant === 'filled' ? 'text-white' : (__VLS_ctx.isColorUp ? 'text-green-500' : 'text-red-500'),
        ]) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
/** @type {__VLS_StyleScopedClasses['transition-transform']} */ ;
/** @type {__VLS_StyleScopedClasses['duration-300']} */ ;
// @ts-ignore
[sizeClasses, size, size, size, variant, variant, variant, variant, variant, isColorUp, isColorUp, isColorUp, isColorUp, isColorUp, prefix, prefix, value, suffix, suffix, isUp, isUp,];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
export default {};
