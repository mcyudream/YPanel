const columns = [
    {
        accessorKey: 'name',
        header: '商品',
        width: 220,
    },
    {
        accessorKey: 'category',
        header: '分类',
        width: 120,
    },
    {
        accessorKey: 'price',
        header: '价格',
        align: 'right',
        width: 120,
    },
    {
        accessorKey: 'stock',
        header: '库存',
        align: 'right',
        width: 120,
    },
    {
        accessorKey: 'status',
        header: '状态',
        width: 120,
    },
];
const data = [
    { id: 'sku-001', name: '协作空间专业版', category: '软件', price: 1280, stock: 42, status: 'available' },
    { id: 'sku-002', name: '团队数据大屏', category: '模板', price: 680, stock: 8, status: 'warning' },
    { id: 'sku-003', name: '年度支持服务', category: '服务', price: 3600, stock: 0, status: 'sold-out' },
];
const statusMap = {
    'available': {
        label: '可售',
        class: 'bg-success/10 text-success',
    },
    'warning': {
        label: '低库存',
        class: 'bg-warning/10 text-warning',
    },
    'sold-out': {
        label: '售罄',
        class: 'bg-muted text-muted-foreground',
    },
};
function formatCurrency(value) {
    return new Intl.NumberFormat('zh-CN', {
        style: 'currency',
        currency: 'CNY',
        maximumFractionDigits: 0,
    }).format(value);
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.FaTable | typeof __VLS_components.FaTable} */
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
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { 'header-name': __VLS_7 } = __VLS_3.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex flex-col" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['flex-col']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "text-xs text-muted-foreground font-normal" },
    });
    /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['font-normal']} */ ;
    // @ts-ignore
    [columns, data,];
}
{
    const { 'cell-name': __VLS_8 } = __VLS_3.slots;
    const [{ row }] = __VLS_vSlot(__VLS_8);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "font-medium" },
    });
    /** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
    (row.original.name);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-xs text-muted-foreground" },
    });
    /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    (row.original.id);
    // @ts-ignore
    [];
}
{
    const { 'cell-price': __VLS_9 } = __VLS_3.slots;
    const [{ value }] = __VLS_vSlot(__VLS_9);
    (__VLS_ctx.formatCurrency(Number(value)));
    // @ts-ignore
    [formatCurrency,];
}
{
    const { 'cell-stock': __VLS_10 } = __VLS_3.slots;
    const [{ value }] = __VLS_vSlot(__VLS_10);
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "tabular-nums" },
    });
    /** @type {__VLS_StyleScopedClasses['tabular-nums']} */ ;
    (value);
    // @ts-ignore
    [];
}
{
    const { 'cell-status': __VLS_11 } = __VLS_3.slots;
    const [{ value }] = __VLS_vSlot(__VLS_11);
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "text-xs font-medium px-2 py-0.5 rounded-full inline-flex" },
        ...{ class: (__VLS_ctx.statusMap[value].class) },
    });
    /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
    /** @type {__VLS_StyleScopedClasses['px-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['py-0.5']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    (__VLS_ctx.statusMap[value].label);
    // @ts-ignore
    [statusMap, statusMap,];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
