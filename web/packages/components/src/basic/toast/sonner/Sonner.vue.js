import { CircleCheckIcon, InfoIcon, Loader2Icon, OctagonXIcon, TriangleAlertIcon, XIcon } from '@lucide/vue';
import { Toaster as Sonner } from 'vue-sonner';
import { cn } from '#utils';
const props = defineProps();
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
/** @ts-ignore @type { | typeof __VLS_components.Sonner | typeof __VLS_components.Sonner} */
Sonner;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ class: (__VLS_ctx.cn('toaster group', props.class)) },
    ...{ style: ({
            '--normal-bg': 'oklch(var(--popover))',
            '--normal-text': 'oklchvar(--popover-foreground))',
            '--normal-border': 'oklchvar(--border))',
            '--border-radius': 'oklchvar(--radius))',
        }) },
    ...(props),
}));
const __VLS_2 = __VLS_1({
    ...{ class: (__VLS_ctx.cn('toaster group', props.class)) },
    ...{ style: ({
            '--normal-bg': 'oklch(var(--popover))',
            '--normal-text': 'oklchvar(--popover-foreground))',
            '--normal-border': 'oklchvar(--border))',
            '--border-radius': 'oklchvar(--radius))',
        }) },
    ...(props),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { 'success-icon': __VLS_7 } = __VLS_3.slots;
    let __VLS_8;
    /** @ts-ignore @type { | typeof __VLS_components.CircleCheckIcon} */
    CircleCheckIcon;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        ...{ class: "size-4" },
    }));
    const __VLS_10 = __VLS_9({
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    // @ts-ignore
    [cn,];
}
{
    const { 'info-icon': __VLS_13 } = __VLS_3.slots;
    let __VLS_14;
    /** @ts-ignore @type { | typeof __VLS_components.InfoIcon} */
    InfoIcon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        ...{ class: "size-4" },
    }));
    const __VLS_16 = __VLS_15({
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    // @ts-ignore
    [];
}
{
    const { 'warning-icon': __VLS_19 } = __VLS_3.slots;
    let __VLS_20;
    /** @ts-ignore @type { | typeof __VLS_components.TriangleAlertIcon} */
    TriangleAlertIcon;
    // @ts-ignore
    const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({
        ...{ class: "size-4" },
    }));
    const __VLS_22 = __VLS_21({
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_21));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    // @ts-ignore
    [];
}
{
    const { 'error-icon': __VLS_25 } = __VLS_3.slots;
    let __VLS_26;
    /** @ts-ignore @type { | typeof __VLS_components.OctagonXIcon} */
    OctagonXIcon;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        ...{ class: "size-4" },
    }));
    const __VLS_28 = __VLS_27({
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    // @ts-ignore
    [];
}
{
    const { 'loading-icon': __VLS_31 } = __VLS_3.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({});
    let __VLS_32;
    /** @ts-ignore @type { | typeof __VLS_components.Loader2Icon} */
    Loader2Icon;
    // @ts-ignore
    const __VLS_33 = __VLS_asFunctionalComponent1(__VLS_32, new __VLS_32({
        ...{ class: "size-4 animate-spin" },
    }));
    const __VLS_34 = __VLS_33({
        ...{ class: "size-4 animate-spin" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_33));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['animate-spin']} */ ;
    // @ts-ignore
    [];
}
{
    const { 'close-icon': __VLS_37 } = __VLS_3.slots;
    let __VLS_38;
    /** @ts-ignore @type { | typeof __VLS_components.XIcon} */
    XIcon;
    // @ts-ignore
    const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({
        ...{ class: "size-4" },
    }));
    const __VLS_40 = __VLS_39({
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_39));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
});
export default {};
