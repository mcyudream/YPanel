import { cn } from '#utils';
defineOptions({
    name: 'BuiltInPageHeader',
});
const props = defineProps();
const slots = defineSlots();
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
    ...{ class: (__VLS_ctx.cn('mb-4 flex flex-wrap items-center justify-between gap-5 border-b bg-background px-5 py-4', props.class)) },
});
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: (__VLS_ctx.cn('flex-[1_1_70%]', props.mainClass)) },
});
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-2xl" },
});
/** @type {__VLS_StyleScopedClasses['text-2xl']} */ ;
__VLS_asFunctionalSlot(slots.title)({});
(__VLS_ctx.title);
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-secondary-foreground/50 mt-2 empty-hidden" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-secondary-foreground/50']} */ ;
/** @type {__VLS_StyleScopedClasses['mt-2']} */ ;
/** @type {__VLS_StyleScopedClasses['empty-hidden']} */ ;
__VLS_asFunctionalSlot(slots.description)({});
(__VLS_ctx.description);
if (!!slots.default) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: (__VLS_ctx.cn('ml-a flex-none', props.defaultClass)) },
    });
    __VLS_asFunctionalSlot(slots['default'])({});
}
// @ts-ignore
[cn, cn, cn, title, description,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
