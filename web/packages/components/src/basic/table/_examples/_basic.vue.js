const columns = [
    {
        accessorKey: 'title',
        header: '任务',
    },
    {
        accessorKey: 'owner',
        header: '负责人',
    },
    {
        accessorKey: 'status',
        header: '状态',
    },
    {
        accessorKey: 'updatedAt',
        header: '更新时间',
    },
];
const data = [
    { id: 't-001', title: '完善组件文档', owner: '林舟', status: '进行中', updatedAt: '2026-05-23' },
    { id: 't-002', title: '同步设计变量', owner: '陈念', status: '待处理', updatedAt: '2026-05-22' },
    { id: 't-003', title: '整理示例数据', owner: '周衡', status: '已完成', updatedAt: '2026-05-21' },
    { id: 't-004', title: '回归交互状态', owner: '沈若', status: '进行中', updatedAt: '2026-05-20' },
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
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}));
const __VLS_2 = __VLS_1({
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[columns, data,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
