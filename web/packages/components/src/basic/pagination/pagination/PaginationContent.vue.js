import { reactiveOmit } from '@vueuse/core';
import { PaginationList } from 'reka-ui';
import { cn } from '#utils';
const props = defineProps();
const delegatedProps = reactiveOmit(props, 'class');
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
/** @ts-ignore @type { | typeof __VLS_components.PaginationList | typeof __VLS_components.PaginationList} */
PaginationList;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "pagination-content",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('flex flex-row items-center gap-1', props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "pagination-content",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn('flex flex-row items-center gap-1', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
{
    const { default: __VLS_6 } = __VLS_3.slots;
    const [slotProps] = __VLS_vSlot(__VLS_6);
    var __VLS_7 = {
        ...(slotProps),
    };
    // @ts-ignore
    [delegatedProps, cn,];
    __VLS_3.slots['' /* empty slot name completion */];
}
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
