const columns = [
    {
        accessorKey: 'name',
        header: '项目',
        width: 220,
    },
    {
        accessorKey: 'owner',
        header: '负责人',
        width: 120,
    },
    {
        accessorKey: 'progress',
        header: '进度',
        width: 120,
        align: 'right',
    },
    {
        accessorKey: 'status',
        header: '状态',
        width: 120,
    },
];
const data = [
    { id: 'p-001', name: '权限策略整理', owner: '林舟', progress: '72%', status: '进行中' },
    { id: 'p-002', name: '表格示例拆分', owner: '陈念', progress: '100%', status: '已完成' },
    { id: 'p-003', name: '数据导出流程', owner: '周衡', progress: '48%', status: '进行中' },
    { id: 'p-004', name: '菜单体验优化', owner: '沈若', progress: '16%', status: '待处理' },
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.FaTable} */
FaTable;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    border: true,
    rowKey: "id",
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}));
const __VLS_2 = __VLS_1({
    border: true,
    rowKey: "id",
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[columns, data,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
