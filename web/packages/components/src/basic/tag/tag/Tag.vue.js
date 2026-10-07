import { Primitive } from 'reka-ui';
import { cn } from '#utils';
import { tagVariants } from '.';
import Icon from '../../icon/index.vue';
const props = withDefaults(defineProps(), {
    as: 'span',
});
const emit = defineEmits();
function handleClose(event) {
    emit('close', event);
}
const __VLS_defaults = {
    as: 'span',
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Primitive | typeof __VLS_components.Primitive} */
Primitive;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "tag",
    as: (__VLS_ctx.as),
    asChild: (__VLS_ctx.asChild),
    ...{ class: (__VLS_ctx.cn(__VLS_ctx.tagVariants({ variant: __VLS_ctx.variant }), props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "tag",
    as: (__VLS_ctx.as),
    asChild: (__VLS_ctx.asChild),
    ...{ class: (__VLS_ctx.cn(__VLS_ctx.tagVariants({ variant: __VLS_ctx.variant }), props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
if (__VLS_ctx.icon) {
    const __VLS_7 = Icon;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
        name: (__VLS_ctx.icon),
        ...{ class: "size-3" },
    }));
    const __VLS_9 = __VLS_8({
        name: (__VLS_ctx.icon),
        ...{ class: "size-3" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_8));
    /** @type {__VLS_StyleScopedClasses['size-3']} */ ;
}
var __VLS_12 = {};
if (__VLS_ctx.closable) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (__VLS_ctx.handleClose) },
        type: "button",
        ...{ class: "me--1 outline-none rounded-full inline-flex ring-offset-background items-center justify-center focus:ring-2 focus:ring-ring focus:ring-offset-2" },
    });
    /** @type {__VLS_StyleScopedClasses['me--1']} */ ;
    /** @type {__VLS_StyleScopedClasses['outline-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['ring-offset-background']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-ring']} */ ;
    /** @type {__VLS_StyleScopedClasses['focus:ring-offset-2']} */ ;
    const __VLS_14 = Icon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        name: "i-lucide:x",
        ...{ class: "size-3" },
    }));
    const __VLS_16 = __VLS_15({
        name: "i-lucide:x",
        ...{ class: "size-3" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    /** @type {__VLS_StyleScopedClasses['size-3']} */ ;
}
// @ts-ignore
[as, asChild, cn, tagVariants, variant, icon, icon, closable, handleClose,];
var __VLS_3;
// @ts-ignore
var __VLS_13 = __VLS_12;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
