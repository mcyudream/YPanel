const columns = [
    {
        accessorKey: 'title',
        header: '事项',
        width: 220,
    },
    {
        accessorKey: 'owner',
        header: '负责人',
        width: 120,
    },
    {
        accessorKey: 'priority',
        header: '优先级',
        width: 120,
    },
    {
        accessorKey: 'status',
        header: '状态',
        width: 120,
    },
    {
        accessorKey: 'updatedAt',
        header: '更新时间',
        width: 160,
    },
];
const data = [
    { id: 't-001', title: '补充表格文档', owner: '林舟', priority: '中', status: '进行中', updatedAt: '2026-05-23' },
    { id: 't-002', title: '优化筛选体验', owner: '陈念', priority: '高', status: '待处理', updatedAt: '2026-05-22' },
    { id: 't-003', title: '同步设计变量', owner: '周衡', priority: '低', status: '已完成', updatedAt: '2026-05-21' },
    { id: 't-004', title: '检查固定列阴影', owner: '沈若', priority: '中', status: '进行中', updatedAt: '2026-05-20' },
    { id: 't-005', title: '整理示例数据', owner: '梁一', priority: '低', status: '已完成', updatedAt: '2026-05-19' },
    { id: 't-006', title: '回归行选择交互', owner: '许知', priority: '高', status: '待处理', updatedAt: '2026-05-18' },
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
    stripe: true,
    rowKey: "id",
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}));
const __VLS_2 = __VLS_1({
    stripe: true,
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
