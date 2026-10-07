import { reactiveOmit } from '@vueuse/core';
import { PaginationListItem } from 'reka-ui';
import { cn } from '#utils';
import { buttonVariants } from '../../button/button';
const props = withDefaults(defineProps(), {
    size: 'icon',
});
const delegatedProps = reactiveOmit(props, 'class', 'size', 'isActive');
const __VLS_defaults = {
    size: 'icon',
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.PaginationListItem | typeof __VLS_components.PaginationListItem} */
PaginationListItem;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    dataSlot: "pagination-item",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn(__VLS_ctx.buttonVariants({
            variant: __VLS_ctx.isActive ? 'outline' : 'ghost',
            size: __VLS_ctx.size,
        }), props.class)) },
}));
const __VLS_2 = __VLS_1({
    dataSlot: "pagination-item",
    ...(__VLS_ctx.delegatedProps),
    ...{ class: (__VLS_ctx.cn(__VLS_ctx.buttonVariants({
            variant: __VLS_ctx.isActive ? 'outline' : 'ghost',
            size: __VLS_ctx.size,
        }), props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
var __VLS_7 = {};
// @ts-ignore
[delegatedProps, cn, buttonVariants, isActive, size,];
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
