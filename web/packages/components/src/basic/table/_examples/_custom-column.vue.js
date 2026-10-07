const columns = [
    {
        accessorKey: 'name',
        label: '指标名称',
        width: 180,
        cellClass: 'font-medium bg-amber-200/80 dark:bg-amber-500/35',
    },
    {
        accessorKey: 'today',
        header: '今日',
        align: 'right',
        width: 120,
        cellClass: 'tabular-nums',
    },
    {
        accessorKey: 'yesterday',
        header: '昨日',
        align: 'right',
        width: 120,
        cellClass: 'tabular-nums text-muted-foreground',
    },
    {
        accessorKey: 'trend',
        title: '变化',
        align: 'right',
        width: 120,
        headerClass: 'text-primary',
        cellClass: ({ row }) => row.original.trend.startsWith('+')
            ? 'text-success font-medium tabular-nums'
            : 'text-destructive font-medium tabular-nums',
    },
];
const data = [
    { id: 'm-001', name: '访问量', today: '18,420', yesterday: '16,280', trend: '+13.1%' },
    { id: 'm-002', name: '转化率', today: '8.6%', yesterday: '9.1%', trend: '-0.5%' },
    { id: 'm-003', name: '新增客户', today: '326', yesterday: '304', trend: '+7.2%' },
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
    rowKey: "id",
    columns: __VLS_ctx.columns,
    data: __VLS_ctx.data,
}));
const __VLS_2 = __VLS_1({
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
