import { reactiveOmit } from '@vueuse/core';
import { useForwardProps } from 'reka-ui';
import { computed } from 'vue';
import { useVueOTPContext } from 'vue-input-otp';
import { cn } from '#utils';
const props = defineProps();
const delegatedProps = reactiveOmit(props, 'class');
const forwarded = useForwardProps(delegatedProps);
const context = useVueOTPContext();
const slot = computed(() => context?.value.slots[props.index]);
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
    ...(__VLS_ctx.forwarded),
    'data-slot': "input-otp-slot",
    'data-active': (__VLS_ctx.slot?.isActive),
    ...{ class: (__VLS_ctx.cn('data-[active=true]:border-ring data-[active=true]:ring-ring/50 data-[active=true]:aria-invalid:ring-destructive/20 dark:data-[active=true]:aria-invalid:ring-destructive/40 aria-invalid:border-destructive data-[active=true]:aria-invalid:border-destructive dark:bg-input/30 border-input relative flex h-9 w-9 items-center justify-center border-y border-e text-sm shadow-xs transition-all outline-none first:rounded-ss-md first:rounded-es-md first:border-s last:rounded-se-md last:rounded-ee-md data-[active=true]:z-10 data-[active=true]:ring-3', props.class)) },
});
(__VLS_ctx.slot?.char);
if (__VLS_ctx.slot?.hasFakeCaret) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex pointer-events-none items-center inset-0 justify-center absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['pointer-events-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div)({
        ...{ class: "animate-caret-blink bg-foreground h-4 w-px duration-1000" },
    });
    /** @type {__VLS_StyleScopedClasses['animate-caret-blink']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['h-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['w-px']} */ ;
    /** @type {__VLS_StyleScopedClasses['duration-1000']} */ ;
}
// @ts-ignore
[forwarded, slot, slot, slot, cn,];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
});
export default {};
