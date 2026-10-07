import { computed, shallowRef } from 'vue';
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
        accessorKey: 'priority',
        header: '优先级',
        enableSorting: true,
    },
    {
        accessorKey: 'updatedAt',
        header: '更新时间',
        enableSorting: true,
    },
];
const data = [
    { id: 't-001', title: '完善组件文档', owner: '林舟', priority: '中', updatedAt: '2026-05-23' },
    { id: 't-002', title: '同步设计变量', owner: '陈念', priority: '高', updatedAt: '2026-05-22' },
    { id: 't-003', title: '整理示例数据', owner: '周衡', priority: '低', updatedAt: '2026-05-21' },
    { id: 't-004', title: '回归交互状态', owner: '沈若', priority: '中', updatedAt: '2026-05-20' },
];
const priorityRank = {
    高: 3,
    中: 2,
    低: 1,
};
const sorting = shallowRef([]);
function getSortValue(row, key) {
    if (key === 'priority') {
        return priorityRank[row.priority];
    }
    return row[key];
}
function compareSortValue(a, b) {
    if (typeof a === 'number' && typeof b === 'number') {
        return a - b;
    }
    return String(a).localeCompare(String(b), 'zh-CN', { numeric: true });
}
const sortedData = computed(() => {
    if (!sorting.value.length) {
        return data;
    }
    return [...data].sort((a, b) => {
        for (const sort of sorting.value) {
            const key = sort.id;
            const aVal = getSortValue(a, key);
            const bVal = getSortValue(b, key);
            if (aVal === bVal) {
                continue;
            }
            const result = compareSortValue(aVal, bVal);
            return sort.desc ? -result : result;
        }
        return 0;
    });
});
const sortingText = computed(() => {
    if (!sorting.value.length) {
        return '暂无排序';
    }
    return sorting.value.map(item => `${item.id}: ${item.desc ? '降序' : '升序'}`).join('，');
});
function handleSortingChange(value) {
    sorting.value = value;
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
    ...{ 'onSortingChange': {} },
    sortable: true,
    rowKey: "id",
    sorting: __VLS_ctx.sorting,
    columns: __VLS_ctx.columns,
    data: (__VLS_ctx.sortedData),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onSortingChange': {} },
    sortable: true,
    rowKey: "id",
    sorting: __VLS_ctx.sorting,
    columns: __VLS_ctx.columns,
    data: (__VLS_ctx.sortedData),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.sortingChange} */
    onSortingChange: (__VLS_ctx.handleSortingChange),
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
(__VLS_ctx.sortingText);
// @ts-ignore
[sorting, columns, sortedData, handleSortingChange, sortingText,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
