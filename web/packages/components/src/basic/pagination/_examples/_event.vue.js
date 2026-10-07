import { ref } from 'vue';
import FaPagination from '../index.vue';
const page = ref(1);
const size = ref(10);
const total = ref(100);
const message = ref('等待分页操作');
function handlePageChange(value) {
    message.value = `当前页码切换为：${value}`;
}
function handleSizeChange(value) {
    message.value = `每页条数切换为：${value}`;
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "space-y-3" },
});
/** @type {__VLS_StyleScopedClasses['space-y-3']} */ ;
const __VLS_0 = FaPagination;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onPageChange': {} },
    ...{ 'onSizeChange': {} },
    page: (__VLS_ctx.page),
    size: (__VLS_ctx.size),
    total: (__VLS_ctx.total),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onPageChange': {} },
    ...{ 'onSizeChange': {} },
    page: (__VLS_ctx.page),
    size: (__VLS_ctx.size),
    total: (__VLS_ctx.total),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.pageChange} */
    onPageChange: (__VLS_ctx.handlePageChange),
};
const __VLS_7 = {
    /** @type {typeof __VLS_5.sizeChange} */
    onSizeChange: (__VLS_ctx.handleSizeChange),
};
var __VLS_3;
var __VLS_4;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
(__VLS_ctx.message);
// @ts-ignore
[page, size, total, handlePageChange, handleSizeChange, message,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
