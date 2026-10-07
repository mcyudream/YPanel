import { ref } from 'vue';
import FaPagination from '../index.vue';
const page = ref(1);
const size = ref(10);
const total = ref(100);
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "space-y-4" },
});
/** @type {__VLS_StyleScopedClasses['space-y-4']} */ ;
const __VLS_0 = FaPagination;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    page: (__VLS_ctx.page),
    size: (__VLS_ctx.size),
    total: (__VLS_ctx.total),
    layout: "jumper, pager, ->, total, sizes",
}));
const __VLS_2 = __VLS_1({
    page: (__VLS_ctx.page),
    size: (__VLS_ctx.size),
    total: (__VLS_ctx.total),
    layout: "jumper, pager, ->, total, sizes",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const __VLS_5 = FaPagination;
// @ts-ignore
const __VLS_6 = __VLS_asFunctionalComponent1(__VLS_5, new __VLS_5({
    page: (__VLS_ctx.page),
    size: (__VLS_ctx.size),
    total: (__VLS_ctx.total),
    layout: "pager",
}));
const __VLS_7 = __VLS_6({
    page: (__VLS_ctx.page),
    size: (__VLS_ctx.size),
    total: (__VLS_ctx.total),
    layout: "pager",
}, ...__VLS_functionalComponentArgsRest(__VLS_6));
// @ts-ignore
[page, page, size, size, total, total,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
