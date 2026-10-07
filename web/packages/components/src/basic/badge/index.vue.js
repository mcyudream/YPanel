import { cva } from 'class-variance-authority';
import { computed, ref } from 'vue';
import { cn } from '#utils';
import { Badge } from './badge';
defineOptions({
    name: 'BuiltInBadge',
});
const props = defineProps();
const badgeDotVariant = cva('absolute start-[100%] h-1.5 w-1.5 rounded-full px-0 ring-1 ring-background before:(absolute inset-0 bg-inherit block h-full w-full animate-ping rounded-full content-empty) -translate-x-[50%] -translate-y-[50%] rtl:(translate-x-[50%]) -indent-9999', {
    variants: {
        variant: {
            default: 'bg-primary hover:bg-primary/80',
            secondary: 'bg-secondary hover:bg-secondary/80',
            destructive: 'bg-destructive hover:bg-destructive/80',
        },
    },
    defaultVariants: {
        variant: 'default',
    },
});
const show = computed(() => {
    switch (typeof props.value) {
        case 'string':
            return props.value.length > 0;
        case 'number':
            return props.value > 0;
        case 'boolean':
            return props.value;
        default:
            return props.value !== undefined && props.value !== null;
    }
});
const transitionClass = ref({
    enterActiveClass: 'ease-in-out duration-500',
    enterFromClass: 'opacity-0',
    enterToClass: 'opacity-100',
    leaveActiveClass: 'ease-in-out duration-500',
    leaveFromClass: 'opacity-100',
    leaveToClass: 'opacity-0',
});
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
    ...{ class: (__VLS_ctx.cn('relative inline-flex', props.class)) },
});
var __VLS_0 = {};
let __VLS_2;
/** @ts-ignore @type { | typeof __VLS_components.Transition | typeof __VLS_components.Transition} */
Transition;
// @ts-ignore
const __VLS_3 = __VLS_asFunctionalComponent1(__VLS_2, new __VLS_2({
    ...(__VLS_ctx.transitionClass),
}));
const __VLS_4 = __VLS_3({
    ...(__VLS_ctx.transitionClass),
}, ...__VLS_functionalComponentArgsRest(__VLS_3));
const { default: __VLS_7 } = __VLS_5.slots;
if (__VLS_ctx.show) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "h-full w-full absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['h-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    if (__VLS_ctx.value === true) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div)({
            ...{ class: (__VLS_ctx.badgeDotVariant({ variant: __VLS_ctx.variant })) },
        });
    }
    else {
        let __VLS_8;
        /** @ts-ignore @type { | typeof __VLS_components.Badge | typeof __VLS_components.Badge} */
        Badge;
        // @ts-ignore
        const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
            variant: __VLS_ctx.variant,
            ...{ class: (__VLS_ctx.cn('absolute start-[50%] top-0 z-20 whitespace-nowrap px-1.5 py-0 ring-1 ring-primary-foreground -translate-y-[50%] hover:bg-none!', props.badgeClass)) },
        }));
        const __VLS_10 = __VLS_9({
            variant: __VLS_ctx.variant,
            ...{ class: (__VLS_ctx.cn('absolute start-[50%] top-0 z-20 whitespace-nowrap px-1.5 py-0 ring-1 ring-primary-foreground -translate-y-[50%] hover:bg-none!', props.badgeClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_9));
        const { default: __VLS_13 } = __VLS_11.slots;
        (__VLS_ctx.value);
        // @ts-ignore
        [cn, cn, transitionClass, show, value, value, badgeDotVariant, variant, variant,];
        var __VLS_11;
    }
}
// @ts-ignore
[];
var __VLS_5;
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
