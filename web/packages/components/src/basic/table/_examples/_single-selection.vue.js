import { computed, shallowRef } from 'vue';
const columns = [
    {
        type: 'selection',
        fixed: 'left',
        width: 50,
        disabled: row => !row.enabled,
    },
    {
        accessorKey: 'name',
        header: '成员',
    },
    {
        accessorKey: 'role',
        header: '角色',
        width: 120,
    },
    {
        accessorKey: 'team',
        header: '团队',
        width: 140,
    },
    {
        accessorKey: 'enabled',
        header: '是否可选',
        width: 120,
    },
];
const data = [
    { id: 'm-001', name: '沈若', role: '管理员', team: '平台组', enabled: true },
    { id: 'm-002', name: '梁一', role: '开发者', team: '体验组', enabled: true },
    { id: 'm-003', name: '许知', role: '访客', team: '运营组', enabled: false },
    { id: 'm-004', name: '苏眠', role: '审计员', team: '风控组', enabled: true },
];
const selectedRows = shallowRef([]);
const selectedName = computed(() => selectedRows.value[0]?.name || '暂无');
function handleSelectionChange(rows) {
    selectedRows.value = rows;
}
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.FaTable} */
FaTable;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onSelectionChange': {} },
    selectable: true,
    rowKey: "id",
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}));
const __VLS_2 = __VLS_1({
    ...{ 'onSelectionChange': {} },
    selectable: true,
    rowKey: "id",
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.selectionChange} */
    onSelectionChange: (__VLS_ctx.handleSelectionChange),
};
var __VLS_3;
var __VLS_4;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground px-4 py-3 rounded-md bg-muted" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
/** @type {__VLS_StyleScopedClasses['px-4']} */ ;
/** @type {__VLS_StyleScopedClasses['py-3']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted']} */ ;
(__VLS_ctx.selectedName);
// @ts-ignore
[columns, data, handleSelectionChange, selectedName,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
