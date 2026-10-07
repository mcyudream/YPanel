import { cn } from '#utils';
defineOptions({
    name: 'BuiltInDivider',
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
    ...{ class: (__VLS_ctx.cn('my-4 w-full flex-center whitespace-nowrap text-sm font-500 after:(h-px w-full min-w-4 bg-border content-empty) before:(h-px w-full min-w-4 bg-border content-empty)', {
            'before:(flex-basis-0)': __VLS_ctx.position === 'start',
            'after:(flex-basis-0)': __VLS_ctx.position === 'end',
            'gap-4': !!slots.default,
        }, props.class)) },
});
__VLS_asFunctionalSlot(slots['default'])({});
// @ts-ignore
[cn, position, position,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
