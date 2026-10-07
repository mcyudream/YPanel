import { columnOrderingFeature, columnPinningFeature, columnSizingFeature, columnVisibilityFeature, createExpandedRowModel, FlexRender, rowExpandingFeature, rowSelectionFeature, rowSortingFeature, tableFeatures, useTable, } from '@tanstack/vue-table';
import { computed, ref, watch } from 'vue';
import { cn } from '#utils';
import Button from '../../basic/button/index.vue';
import { Checkbox } from '../../basic/checkbox/checkbox';
import Dropdown from '../../basic/dropdown/index.vue';
import Icon from '../../basic/icon/index.vue';
import { RadioGroup, RadioGroupItem } from '../../basic/radio-group/radio-group';
import { Table, TableBody, TableCaption, TableCell, TableEmpty, TableHead, TableHeader, TableRow, } from './table';
const SELECTION_COLUMN_ID = '__fa_table_selection__';
const TABLE_HEADER_ROW_HEIGHT = 40;
const faTableFeatures = tableFeatures({
    columnOrderingFeature,
    columnSizingFeature,
    columnPinningFeature,
    columnVisibilityFeature,
    rowExpandingFeature,
    expandedRowModel: createExpandedRowModel(),
    rowSelectionFeature,
    rowSortingFeature,
});
export default {};
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInTable',
    });
    const props = withDefaults(defineProps(), {
        border: false,
        caption: undefined,
        columnVisibility: false,
        defaultExpanded: undefined,
        enableMultiSort: true,
        enableSortingRemoval: true,
        expanded: undefined,
        emptyText: '暂无数据',
        indentSize: 20,
        manualExpanding: false,
        multiple: false,
        selectable: false,
        sortable: false,
        sortDescFirst: true,
        stripe: false,
        tree: false,
    });
    const emit = defineEmits();
    const slots = defineSlots();
    const rowSelection = ref({});
    const columnVisibilityState = ref({});
    const sortingState = ref(props.sorting ?? props.defaultSorting ?? []);
    const expandedState = ref(props.expanded ?? props.defaultExpanded ?? {});
    const isSortingControlled = computed(() => props.sorting !== undefined);
    const isExpandedControlled = computed(() => props.expanded !== undefined);
    function isSelectionColumnDef(column) {
        return column.type === 'selection';
    }
    function normalizeFixed(value) {
        if (value === true) {
            return 'start';
        }
        if (value === 'left') {
            return 'start';
        }
        if (value === 'right') {
            return 'end';
        }
        return undefined;
    }
    function parseNumericWidth(width) {
        if (typeof width === 'number') {
            return width;
        }
        if (typeof width === 'string') {
            const matched = width.trim().match(/^(\d+(?:\.\d+)?)px$/);
            return matched ? Number(matched[1]) : undefined;
        }
        return undefined;
    }
    function getColumnId(column) {
        if (isSelectionColumnDef(column)) {
            return column.id ? String(column.id) : SELECTION_COLUMN_ID;
        }
        if (column.id) {
            return String(column.id);
        }
        if ('accessorKey' in column && column.accessorKey) {
            return String(column.accessorKey).replaceAll('.', '_');
        }
        if (typeof column.header === 'string') {
            return column.header;
        }
        return undefined;
    }
    function normalizeColumn(column) {
        const normalized = { ...column };
        if (isSelectionColumnDef(normalized)) {
            normalized.id ??= SELECTION_COLUMN_ID;
            normalized.header ??= '';
            normalized.cell ??= '';
            normalized.align ??= 'center';
            normalized.enableHiding ??= false;
            normalized.enableSorting ??= false;
            normalized.size ??= parseNumericWidth(normalized.width) ?? 40;
            normalized.minSize ??= parseNumericWidth(normalized.minWidth);
            normalized.maxSize ??= parseNumericWidth(normalized.maxWidth);
            normalized.width ??= normalized.size;
            return normalized;
        }
        normalized.header ??= normalized.label ?? normalized.title;
        normalized.enableSorting ??= false;
        normalized.size ??= parseNumericWidth(normalized.width);
        normalized.minSize ??= parseNumericWidth(normalized.minWidth);
        normalized.maxSize ??= parseNumericWidth(normalized.maxWidth);
        if (normalized.columns?.length) {
            normalized.columns = normalized.columns.map(item => normalizeColumn(item));
        }
        return normalized;
    }
    const normalizedColumns = computed(() => props.columns.map(column => normalizeColumn(column)));
    const resolvedColumns = computed(() => normalizedColumns.value);
    function findSelectionColumn(columns) {
        for (const column of columns) {
            if (isSelectionColumnDef(column)) {
                return column;
            }
            if (column.columns?.length) {
                const selectionColumn = findSelectionColumn(column.columns);
                if (selectionColumn) {
                    return selectionColumn;
                }
            }
        }
    }
    const selectionColumnDef = computed(() => findSelectionColumn(normalizedColumns.value));
    function getColumnPinningState(columns) {
        const start = new Set();
        const end = new Set();
        function collect(items, inheritedFixed) {
            items.forEach((column) => {
                const fixed = normalizeFixed(column.fixed) ?? inheritedFixed;
                if (column.columns?.length) {
                    collect(column.columns, fixed);
                    return;
                }
                const columnId = getColumnId(column);
                if (!columnId || !fixed) {
                    return;
                }
                if (fixed === 'start') {
                    start.add(columnId);
                    end.delete(columnId);
                }
                else {
                    end.add(columnId);
                    start.delete(columnId);
                }
            });
        }
        collect(columns);
        return {
            start: [...start],
            end: [...end],
        };
    }
    const columnPinning = computed(() => getColumnPinningState(normalizedColumns.value));
    function resolveRowId(row, index, parent) {
        if (props.getRowId) {
            return props.getRowId(row, index, parent);
        }
        if (typeof props.rowKey === 'function') {
            return String(props.rowKey(row, index));
        }
        if (props.rowKey) {
            const value = row[String(props.rowKey)];
            if (value != null) {
                return String(value);
            }
        }
        return parent ? `${parent.id}.${index}` : String(index);
    }
    function resolveSubRows(row, index) {
        if (!props.tree) {
            return undefined;
        }
        if (props.getSubRows) {
            return props.getSubRows(row, index);
        }
        const children = row.children;
        return Array.isArray(children) ? children : undefined;
    }
    function isRowSelectionDisabled(row) {
        return selectionColumnDef.value?.disabled?.(row.original, row.index) ?? false;
    }
    function syncSingleRowSelection() {
        if (props.multiple) {
            return;
        }
        const selectedIds = Object.keys(rowSelection.value).filter(id => rowSelection.value[id]);
        if (selectedIds.length <= 1) {
            return;
        }
        const selectedId = selectedIds.at(-1);
        rowSelection.value = selectedId ? { [selectedId]: true } : {};
    }
    function handleRowSelectionChange(updaterOrValue) {
        rowSelection.value = typeof updaterOrValue === 'function'
            ? updaterOrValue(rowSelection.value)
            : updaterOrValue;
        syncSingleRowSelection();
    }
    function handleColumnVisibilityChange(updaterOrValue) {
        columnVisibilityState.value = typeof updaterOrValue === 'function'
            ? updaterOrValue(columnVisibilityState.value)
            : updaterOrValue;
    }
    function handleSortingChange(updaterOrValue) {
        const nextSorting = typeof updaterOrValue === 'function'
            ? updaterOrValue(sortingState.value)
            : updaterOrValue;
        if (!isSortingControlled.value) {
            sortingState.value = nextSorting;
        }
        emit('sortingChange', nextSorting, table);
    }
    function handleExpandedChange(updaterOrValue) {
        const nextExpanded = typeof updaterOrValue === 'function'
            ? updaterOrValue(expandedState.value)
            : updaterOrValue;
        if (!isExpandedControlled.value) {
            expandedState.value = nextExpanded;
        }
        emit('update:expanded', nextExpanded);
        emit('expandedChange', nextExpanded, table);
    }
    const tableState = computed(() => ({
        columnVisibility: columnVisibilityState.value,
        columnPinning: columnPinning.value,
        rowSelection: rowSelection.value,
        expanded: expandedState.value,
        sorting: sortingState.value,
    }));
    const table = useTable({
        features: faTableFeatures,
        data: computed(() => props.data),
        get enableMultiSort() {
            return props.enableMultiSort;
        },
        get enableMultiRowSelection() {
            return props.multiple;
        },
        get enableSorting() {
            return props.sortable;
        },
        get enableSortingRemoval() {
            return props.enableSortingRemoval;
        },
        enableRowSelection: row => props.selectable && Boolean(selectionColumnDef.value) && !isRowSelectionDisabled(row),
        get columns() {
            return resolvedColumns.value;
        },
        getRowId: resolveRowId,
        getSubRows: resolveSubRows,
        get manualExpanding() {
            return props.manualExpanding;
        },
        manualSorting: true,
        onColumnVisibilityChange: handleColumnVisibilityChange,
        onRowSelectionChange: handleRowSelectionChange,
        onExpandedChange: handleExpandedChange,
        onSortingChange: handleSortingChange,
        get sortDescFirst() {
            return props.sortDescFirst;
        },
        state: tableState,
    });
    const selectedRows = computed(() => table.getSelectedRowModel().rows.map(row => row.original));
    const tableRows = computed(() => {
        void tableState.value;
        return table.getRowModel().rows;
    });
    const visibleColumnCount = computed(() => {
        void tableState.value.columnVisibility;
        return Math.max(table.getVisibleLeafColumns().length, 1);
    });
    const isSingleSelectionMode = computed(() => props.selectable && !props.multiple && Boolean(selectionColumnDef.value));
    const singleSelectedRowId = computed(() => Object.keys(rowSelection.value).find(id => rowSelection.value[id]));
    const tableRootComponent = computed(() => isSingleSelectionMode.value ? RadioGroup : 'div');
    const tableRootModelValue = computed(() => isSingleSelectionMode.value ? singleSelectedRowId.value : undefined);
    const stripeInteractionClass = computed(() => props.stripe ? '[&:hover>td]:bg-muted/50 [&[data-state=selected]>td]:bg-muted' : undefined);
    const hasToolbar = computed(() => props.columnVisibility || Boolean(slots.toolbar));
    const columnVisibilityColumns = computed(() => {
        void resolvedColumns.value;
        void tableState.value.columnVisibility;
        return table.getAllLeafColumns().filter(column => column.getCanHide());
    });
    const columnVisibilityMenuItems = computed(() => [
        columnVisibilityColumns.value.length
            ? columnVisibilityColumns.value.map(column => ({
                type: 'checkbox',
                label: getColumnVisibilityLabel(column),
                checked: column.getIsVisible(),
                handle: (checked) => column.toggleVisibility(checked),
            }))
            : [],
    ]);
    watch(() => props.multiple, () => {
        syncSingleRowSelection();
    }, { immediate: true });
    watch(() => props.sorting, (sorting) => {
        if (sorting !== undefined) {
            sortingState.value = sorting;
        }
    }, { deep: true });
    watch(() => props.expanded, (expanded) => {
        if (expanded !== undefined) {
            expandedState.value = expanded;
        }
    }, { deep: true });
    watch([rowSelection, () => props.data], () => {
        emit('selectionChange', selectedRows.value, rowSelection.value);
    }, { deep: true });
    function isSelectionColumn(column) {
        return isSelectionColumnDef(column.columnDef);
    }
    function getColumnSlotName(prefix, columnId) {
        return `${prefix}-${columnId}`;
    }
    function getHeaderSlotName(header) {
        return getColumnSlotName('header', header.column.id);
    }
    function getCellSlotName(cell) {
        return getColumnSlotName('cell', cell.column.id);
    }
    function hasColumnSlot(name) {
        return Boolean(slots[name]);
    }
    function getColumnVisibilityLabel(column) {
        const columnDef = column.columnDef;
        if (columnDef.label) {
            return columnDef.label;
        }
        if (columnDef.title) {
            return columnDef.title;
        }
        if (typeof columnDef.header === 'string' && columnDef.header) {
            return columnDef.header;
        }
        return column.id;
    }
    function getHeaderSlotProps(header) {
        return {
            column: header.column,
            header,
            table,
        };
    }
    function getHeaderAriaSort(header) {
        if (!props.sortable || !header.column.getCanSort()) {
            return undefined;
        }
        const sorted = header.column.getIsSorted();
        if (sorted === 'asc') {
            return 'ascending';
        }
        if (sorted === 'desc') {
            return 'descending';
        }
        return undefined;
    }
    function getCellSlotProps(cell, row) {
        return {
            cell,
            column: cell.column,
            index: row.index,
            row,
            table,
            value: cell.getValue(),
        };
    }
    function shouldRenderTreeCell(cell, row) {
        if (!props.tree || isSelectionColumn(cell.column)) {
            return false;
        }
        return row.getVisibleCells().find(item => !isSelectionColumn(item.column))?.id === cell.id;
    }
    function getTreeCellIndentStyle(row) {
        return {
            paddingLeft: `${row.depth * props.indentSize}px`,
        };
    }
    function getTreeToggleIconName(row) {
        return row.getIsExpanded() ? 'i-lucide:chevron-down' : 'i-lucide:chevron-right';
    }
    function handleToggleRowExpanded(row) {
        row.toggleExpanded();
    }
    function getAlignClass(align) {
        if (align === 'center') {
            return 'text-center';
        }
        if (align === 'right') {
            return 'text-right';
        }
        return undefined;
    }
    function resolveSize(size) {
        if (size == null || size === '') {
            return undefined;
        }
        return typeof size === 'number' ? `${size}px` : size;
    }
    function getColumnStyle(column) {
        const columnDef = column.columnDef;
        const pin = column.getIsPinned();
        const width = resolveSize(columnDef.width ?? (pin ? column.getSize() : columnDef.size && columnDef.size !== 150 ? columnDef.size : undefined));
        const minSize = columnDef.minSize && columnDef.minSize !== 20 ? columnDef.minSize : undefined;
        const maxSize = columnDef.maxSize && columnDef.maxSize !== Number.MAX_SAFE_INTEGER ? columnDef.maxSize : undefined;
        const minWidth = resolveSize(columnDef.minWidth ?? minSize);
        const maxWidth = resolveSize(columnDef.maxWidth ?? maxSize);
        const style = {};
        if (width) {
            style.width = width;
            style.minWidth = minWidth ?? width;
        }
        else if (minWidth) {
            style.minWidth = minWidth;
        }
        if (maxWidth) {
            style.maxWidth = maxWidth;
        }
        if (pin === 'start') {
            style.insetInlineStart = `${column.getStart('start')}px`;
            style.position = 'sticky';
        }
        else if (pin === 'end') {
            style.position = 'sticky';
            style.insetInlineEnd = `${column.getAfter('end')}px`;
        }
        if (!Object.keys(style).length) {
            return undefined;
        }
        return style;
    }
    function resolveColumnClass(columnClass, context) {
        return typeof columnClass === 'function' ? columnClass(context) : columnClass;
    }
    function getHeaderClass(header) {
        const columnDef = header.column.columnDef;
        return cn('sticky z-2 bg-muted text-muted-foreground after:pointer-events-none after:absolute after:inset-x-0 after:bottom-0 after:h-px after:bg-border after:content-empty', getAlignClass(columnDef.align), getHeaderBorderClass(header), getPinnedColumnClass(header.column, 'header'), header.column.getCanSort() && 'cursor-pointer select-none', getSortedColumnClass(header.column, 'header'), columnDef.class, resolveColumnClass(columnDef.headerClass, header.getContext()), isSelectionColumn(header.column) && cn('px-3', props.selectionColumnClass));
    }
    function getHeaderStyle(header, headerGroupIndex) {
        return {
            ...getColumnStyle(header.column),
            top: `${headerGroupIndex * TABLE_HEADER_ROW_HEIGHT}px`,
        };
    }
    function getHeaderContentClass(header) {
        const columnDef = header.column.columnDef;
        return cn('inline-flex w-full items-center gap-1.5', columnDef.align === 'center' && 'justify-center', columnDef.align === 'right' && 'justify-end');
    }
    function getCellClass(cell) {
        const columnDef = cell.column.columnDef;
        return cn(getAlignClass(columnDef.align), getCellBorderClass(cell), getPinnedColumnClass(cell.column, 'cell'), getSortedColumnClass(cell.column, 'cell'), columnDef.class, resolveColumnClass(columnDef.cellClass, cell.getContext()), isSelectionColumn(cell.column) && cn('px-3', props.selectionColumnClass));
    }
    function getSortedColumnClass(column, area) {
        if (!props.sortable || !column.getIsSorted()) {
            return undefined;
        }
        return area === 'header'
            ? '!bg-primary/20 text-foreground'
            : '!bg-primary/10';
    }
    function getPinnedColumnClass(column, area) {
        const pin = column.getIsPinned();
        if (!pin) {
            return undefined;
        }
        return cn('sticky', area === 'header' ? 'z-3' : 'z-1', area === 'cell' && 'bg-background transition-colors', pin === 'start' && column.getIsLastColumn('start') && 'shadow-[4px_0_6px_-4px_rgb(0_0_0/0.18)]', pin === 'end' && column.getIsFirstColumn('end') && 'shadow-[-4px_0_6px_-4px_rgb(0_0_0/0.18)]');
    }
    function isLastVisibleLeafColumn(column) {
        return table.getVisibleLeafColumns().at(-1)?.id === column.id;
    }
    function getNextVisibleLeafColumn(column) {
        const columns = table.getVisibleLeafColumns();
        const columnIndex = columns.findIndex(item => item.id === column.id);
        return columnIndex === -1 ? undefined : columns[columnIndex + 1];
    }
    function getVisiblePinnedColumns(pin) {
        return table.getPinnedVisibleLeafColumns(pin);
    }
    function isFirstVisiblePinnedColumn(column, pin) {
        return getVisiblePinnedColumns(pin).at(0)?.id === column.id;
    }
    function isLastVisiblePinnedColumn(column, pin) {
        return getVisiblePinnedColumns(pin).at(-1)?.id === column.id;
    }
    function getHeaderBorderClass(header) {
        if (!props.border) {
            return undefined;
        }
        const leafColumns = header.column.getLeafColumns();
        const lastLeafColumn = leafColumns.at(-1) ?? header.column;
        const pin = lastLeafColumn.getIsPinned();
        if (pin === 'start') {
            return cn(!isFirstVisiblePinnedColumn(lastLeafColumn, 'start')
                && 'before:pointer-events-none before:absolute before:inset-y-0 before:left-0 before:z-1 before:w-px before:bg-border before:content-empty', isLastVisiblePinnedColumn(lastLeafColumn, 'start') && !isLastVisibleLeafColumn(lastLeafColumn)
                && 'bg-[linear-gradient(to_left,oklch(var(--border))_1px,transparent_1px)]');
        }
        if (pin === 'end') {
            return 'before:pointer-events-none before:absolute before:inset-y-0 before:left-0 before:z-1 before:w-px before:bg-border before:content-empty';
        }
        if (getNextVisibleLeafColumn(lastLeafColumn)?.getIsPinned() === 'end') {
            return undefined;
        }
        return isLastVisibleLeafColumn(lastLeafColumn) ? undefined : 'border-r';
    }
    function getCellBorderClass(cell) {
        if (!props.border) {
            return undefined;
        }
        const pin = cell.column.getIsPinned();
        if (pin === 'start') {
            return cn(!isFirstVisiblePinnedColumn(cell.column, 'start')
                && 'before:pointer-events-none before:absolute before:inset-y-0 before:left-0 before:z-1 before:w-px before:bg-border before:content-empty', isLastVisiblePinnedColumn(cell.column, 'start') && !isLastVisibleLeafColumn(cell.column)
                && 'after:pointer-events-none after:absolute after:inset-y-0 after:right-0 after:z-1 after:w-px after:bg-border after:content-empty');
        }
        if (pin === 'end') {
            return 'after:pointer-events-none after:absolute after:inset-y-0 after:left-0 after:z-1 after:w-px after:bg-border after:content-empty';
        }
        if (getNextVisibleLeafColumn(cell.column)?.getIsPinned() === 'end') {
            return undefined;
        }
        return isLastVisibleLeafColumn(cell.column) ? undefined : 'border-r';
    }
    function getRowClass(row, rowIndex) {
        return cn('bg-background', '[&:hover>td[data-pinned]]:bg-muted [&[data-state=selected]>td[data-pinned]]:bg-muted', props.stripe && rowIndex % 2 === 1 && '[&>td]:bg-muted/50', props.stripe && rowIndex % 2 === 1 && '[&>td[data-pinned]]:bg-muted', stripeInteractionClass.value, typeof props.rowClass === 'function' ? props.rowClass(row) : props.rowClass);
    }
    function getHeaderCheckboxValue() {
        if (table.getIsAllRowsSelected()) {
            return true;
        }
        if (table.getIsSomeRowsSelected() && !table.getIsAllRowsSelected()) {
            return 'indeterminate';
        }
        return false;
    }
    function shouldRenderSelectionHeader(header) {
        return props.selectable && props.multiple && isSelectionColumn(header.column) && !header.column.columnDef.header;
    }
    function getSortIconName(header) {
        const sorted = header.column.getIsSorted();
        if (sorted === 'desc') {
            return 'i-lucide:arrow-down';
        }
        if (sorted === 'asc') {
            return 'i-lucide:arrow-up';
        }
        return 'i-lucide:arrow-up-down';
    }
    function getSortIconClass(header) {
        return cn('shrink-0 size-3.5', header.column.getIsSorted() ? 'text-primary' : 'text-muted-foreground/80');
    }
    function shouldRenderSortIndex(header) {
        return props.enableMultiSort && header.column.getSortIndex() > -1 && sortingState.value.length > 1;
    }
    function getSortIndexLabel(header) {
        return header.column.getSortIndex() + 1;
    }
    function handleHeaderClick(header, event) {
        if (!props.sortable || !header.column.getCanSort()) {
            return;
        }
        header.column.getToggleSortingHandler()?.(event);
    }
    function handleToggleAllRows(value) {
        table.toggleAllRowsSelected(value === true);
    }
    function handleToggleRow(row, value) {
        row.toggleSelected(value === true);
    }
    function handleSingleSelectionChange(value) {
        if (value == null) {
            return;
        }
        rowSelection.value = {
            [String(value)]: true,
        };
    }
    function handleRowClick(row, event) {
        emit('rowClick', row.original, row.index, event);
    }
    const __VLS_exposed = {
        table,
    };
    defineExpose(__VLS_exposed);
    const __VLS_defaults = {
        border: false,
        caption: undefined,
        columnVisibility: false,
        defaultExpanded: undefined,
        enableMultiSort: true,
        enableSortingRemoval: true,
        expanded: undefined,
        emptyText: '暂无数据',
        indentSize: 20,
        manualExpanding: false,
        multiple: false,
        selectable: false,
        sortable: false,
        sortDescFirst: true,
        stripe: false,
        tree: false,
    };
    const __VLS_ctx = {
        ...{},
        ...{},
        ...{},
        ...{},
        ...{},
    };
    let __VLS_components;
    let __VLS_intrinsics;
    let __VLS_directives;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: (__VLS_ctx.cn('flex flex-col gap-2', props.class)) },
    });
    if (__VLS_ctx.hasToolbar) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "flex gap-2 items-center" },
        });
        /** @type {__VLS_StyleScopedClasses['flex']} */ ;
        /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "flex flex-1 gap-2 min-w-0 items-center" },
        });
        /** @type {__VLS_StyleScopedClasses['flex']} */ ;
        /** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['min-w-0']} */ ;
        /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
        __VLS_asFunctionalSlot(slots.toolbar)({
            table: (__VLS_ctx.table),
        });
        if (props.columnVisibility) {
            const __VLS_1 = Dropdown || Dropdown;
            // @ts-ignore
            const __VLS_2 = __VLS_asFunctionalComponent1(__VLS_1, new __VLS_1({
                items: (__VLS_ctx.columnVisibilityMenuItems),
                align: "end",
            }));
            const __VLS_3 = __VLS_2({
                items: (__VLS_ctx.columnVisibilityMenuItems),
                align: "end",
            }, ...__VLS_functionalComponentArgsRest(__VLS_2));
            const { default: __VLS_6 } = __VLS_4.slots;
            const __VLS_7 = Button || Button;
            // @ts-ignore
            const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
                variant: "outline",
                size: "icon",
                disabled: (!__VLS_ctx.columnVisibilityColumns.length),
            }));
            const __VLS_9 = __VLS_8({
                variant: "outline",
                size: "icon",
                disabled: (!__VLS_ctx.columnVisibilityColumns.length),
            }, ...__VLS_functionalComponentArgsRest(__VLS_8));
            const { default: __VLS_12 } = __VLS_10.slots;
            const __VLS_13 = Icon;
            // @ts-ignore
            const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
                name: "i-lucide:columns-3",
            }));
            const __VLS_15 = __VLS_14({
                name: "i-lucide:columns-3",
            }, ...__VLS_functionalComponentArgsRest(__VLS_14));
            // @ts-ignore
            [cn, hasToolbar, table, columnVisibilityMenuItems, columnVisibilityColumns,];
            var __VLS_10;
            // @ts-ignore
            [];
            var __VLS_4;
        }
    }
    const __VLS_18 = (__VLS_ctx.tableRootComponent);
    // @ts-ignore
    const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
        ...{ 'onUpdate:modelValue': {} },
        ...{ class: (__VLS_ctx.cn('size-full', props.border && 'border rounded-lg overflow-hidden', props.tableRootClass)) },
        modelValue: (__VLS_ctx.tableRootModelValue),
    }));
    const __VLS_20 = __VLS_19({
        ...{ 'onUpdate:modelValue': {} },
        ...{ class: (__VLS_ctx.cn('size-full', props.border && 'border rounded-lg overflow-hidden', props.tableRootClass)) },
        modelValue: (__VLS_ctx.tableRootModelValue),
    }, ...__VLS_functionalComponentArgsRest(__VLS_19));
    let __VLS_23;
    const __VLS_24 = {
        /** @type {typeof __VLS_23.'update:modelValue'} */
        'onUpdate:modelValue': (__VLS_ctx.handleSingleSelectionChange),
    };
    const { default: __VLS_25 } = __VLS_21.slots;
    let __VLS_26;
    /** @ts-ignore @type { | typeof __VLS_components.Table | typeof __VLS_components.Table} */
    Table;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        ...{ class: (__VLS_ctx.cn(__VLS_ctx.tableClass, !__VLS_ctx.tableRows.length && 'h-full')) },
    }));
    const __VLS_28 = __VLS_27({
        ...{ class: (__VLS_ctx.cn(__VLS_ctx.tableClass, !__VLS_ctx.tableRows.length && 'h-full')) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    const { default: __VLS_31 } = __VLS_29.slots;
    __VLS_asFunctionalSlot(slots.caption)({});
    if (__VLS_ctx.caption) {
        let __VLS_33;
        /** @ts-ignore @type { | typeof __VLS_components.TableCaption | typeof __VLS_components.TableCaption} */
        TableCaption;
        // @ts-ignore
        const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({}));
        const __VLS_35 = __VLS_34({}, ...__VLS_functionalComponentArgsRest(__VLS_34));
        const { default: __VLS_38 } = __VLS_36.slots;
        (__VLS_ctx.caption);
        // @ts-ignore
        [cn, cn, tableRootComponent, tableRootModelValue, handleSingleSelectionChange, tableClass, tableRows, caption, caption,];
        var __VLS_36;
    }
    let __VLS_39;
    /** @ts-ignore @type { | typeof __VLS_components.TableHeader | typeof __VLS_components.TableHeader} */
    TableHeader;
    // @ts-ignore
    const __VLS_40 = __VLS_asFunctionalComponent1(__VLS_39, new __VLS_39({
        ...{ class: (__VLS_ctx.headerClass) },
    }));
    const __VLS_41 = __VLS_40({
        ...{ class: (__VLS_ctx.headerClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_40));
    const { default: __VLS_44 } = __VLS_42.slots;
    for (const [headerGroup, headerGroupIndex] of __VLS_vFor((__VLS_ctx.table.getHeaderGroups()))) {
        let __VLS_45;
        /** @ts-ignore @type { | typeof __VLS_components.TableRow | typeof __VLS_components.TableRow} */
        TableRow;
        // @ts-ignore
        const __VLS_46 = __VLS_asFunctionalComponent1(__VLS_45, new __VLS_45({
            key: (headerGroup.id),
            ...{ class: (__VLS_ctx.cn('border-b-0', __VLS_ctx.headerRowClass)) },
        }));
        const __VLS_47 = __VLS_46({
            key: (headerGroup.id),
            ...{ class: (__VLS_ctx.cn('border-b-0', __VLS_ctx.headerRowClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_46));
        const { default: __VLS_50 } = __VLS_48.slots;
        for (const [header] of __VLS_vFor((headerGroup.headers))) {
            let __VLS_51;
            /** @ts-ignore @type { | typeof __VLS_components.TableHead | typeof __VLS_components.TableHead} */
            TableHead;
            // @ts-ignore
            const __VLS_52 = __VLS_asFunctionalComponent1(__VLS_51, new __VLS_51({
                ...{ 'onClick': {} },
                key: (header.id),
                colspan: (header.colSpan),
                ...{ class: (__VLS_ctx.getHeaderClass(header)) },
                ...{ style: (__VLS_ctx.getHeaderStyle(header, headerGroupIndex)) },
                'aria-sort': (__VLS_ctx.getHeaderAriaSort(header)),
                dataSortable: (props.sortable && header.column.getCanSort() ? '' : undefined),
            }));
            const __VLS_53 = __VLS_52({
                ...{ 'onClick': {} },
                key: (header.id),
                colspan: (header.colSpan),
                ...{ class: (__VLS_ctx.getHeaderClass(header)) },
                ...{ style: (__VLS_ctx.getHeaderStyle(header, headerGroupIndex)) },
                'aria-sort': (__VLS_ctx.getHeaderAriaSort(header)),
                dataSortable: (props.sortable && header.column.getCanSort() ? '' : undefined),
            }, ...__VLS_functionalComponentArgsRest(__VLS_52));
            let __VLS_56;
            const __VLS_57 = {
                /** @type {typeof __VLS_56.click} */
                onClick: (...[$event]) => {
                    return (__VLS_ctx.handleHeaderClick(header, $event));
                    // @ts-ignore
                    [cn, table, headerClass, headerRowClass, getHeaderClass, getHeaderStyle, getHeaderAriaSort, handleHeaderClick,];
                },
            };
            const { default: __VLS_58 } = __VLS_54.slots;
            if (!header.isPlaceholder) {
                __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                    ...{ class: (__VLS_ctx.getHeaderContentClass(header)) },
                });
                if (__VLS_ctx.hasColumnSlot(__VLS_ctx.getHeaderSlotName(header))) {
                    __VLS_asFunctionalSlot(slots[(__VLS_ctx.getHeaderSlotName(header))])({
                        ...(__VLS_ctx.getHeaderSlotProps(header)),
                    });
                }
                else if (__VLS_ctx.shouldRenderSelectionHeader(header)) {
                    let __VLS_60;
                    /** @ts-ignore @type { | typeof __VLS_components.Checkbox} */
                    Checkbox;
                    // @ts-ignore
                    const __VLS_61 = __VLS_asFunctionalComponent1(__VLS_60, new __VLS_60({
                        ...{ 'onClick': {} },
                        ...{ 'onUpdate:modelValue': {} },
                        modelValue: (__VLS_ctx.getHeaderCheckboxValue()),
                        ...{ class: "translate-y-[2px]" },
                    }));
                    const __VLS_62 = __VLS_61({
                        ...{ 'onClick': {} },
                        ...{ 'onUpdate:modelValue': {} },
                        modelValue: (__VLS_ctx.getHeaderCheckboxValue()),
                        ...{ class: "translate-y-[2px]" },
                    }, ...__VLS_functionalComponentArgsRest(__VLS_61));
                    let __VLS_65;
                    const __VLS_66 = {
                        /** @type {typeof __VLS_65.click} */
                        onClick: () => { },
                    };
                    const __VLS_67 = {
                        /** @type {typeof __VLS_65.'update:modelValue'} */
                        'onUpdate:modelValue': (__VLS_ctx.handleToggleAllRows),
                    };
                    /** @type {__VLS_StyleScopedClasses['translate-y-[2px]']} */ ;
                    var __VLS_63;
                    var __VLS_64;
                }
                else {
                    let __VLS_68;
                    /** @ts-ignore @type { | typeof __VLS_components.FlexRender} */
                    FlexRender;
                    // @ts-ignore
                    const __VLS_69 = __VLS_asFunctionalComponent1(__VLS_68, new __VLS_68({
                        header: (header),
                    }));
                    const __VLS_70 = __VLS_69({
                        header: (header),
                    }, ...__VLS_functionalComponentArgsRest(__VLS_69));
                }
                if (props.sortable && header.column.getCanSort()) {
                    const __VLS_73 = Icon;
                    // @ts-ignore
                    const __VLS_74 = __VLS_asFunctionalComponent1(__VLS_73, new __VLS_73({
                        name: (__VLS_ctx.getSortIconName(header)),
                        ...{ class: (__VLS_ctx.getSortIconClass(header)) },
                    }));
                    const __VLS_75 = __VLS_74({
                        name: (__VLS_ctx.getSortIconName(header)),
                        ...{ class: (__VLS_ctx.getSortIconClass(header)) },
                    }, ...__VLS_functionalComponentArgsRest(__VLS_74));
                    if (__VLS_ctx.shouldRenderSortIndex(header)) {
                        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
                            ...{ class: "text-[10px] text-muted-foreground leading-none font-medium rounded-full bg-muted inline-flex size-4 items-center justify-center" },
                        });
                        /** @type {__VLS_StyleScopedClasses['text-[10px]']} */ ;
                        /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
                        /** @type {__VLS_StyleScopedClasses['leading-none']} */ ;
                        /** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
                        /** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
                        /** @type {__VLS_StyleScopedClasses['bg-muted']} */ ;
                        /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
                        /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
                        /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
                        /** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
                        (__VLS_ctx.getSortIndexLabel(header));
                    }
                }
            }
            // @ts-ignore
            [getHeaderContentClass, hasColumnSlot, getHeaderSlotName, getHeaderSlotName, getHeaderSlotProps, shouldRenderSelectionHeader, getHeaderCheckboxValue, handleToggleAllRows, getSortIconName, getSortIconClass, shouldRenderSortIndex, getSortIndexLabel,];
            var __VLS_54;
            var __VLS_55;
            // @ts-ignore
            [];
        }
        // @ts-ignore
        [];
        var __VLS_48;
        // @ts-ignore
        [];
    }
    // @ts-ignore
    [];
    var __VLS_42;
    let __VLS_78;
    /** @ts-ignore @type { | typeof __VLS_components.TableBody | typeof __VLS_components.TableBody} */
    TableBody;
    // @ts-ignore
    const __VLS_79 = __VLS_asFunctionalComponent1(__VLS_78, new __VLS_78({
        ...{ class: (__VLS_ctx.bodyClass) },
    }));
    const __VLS_80 = __VLS_79({
        ...{ class: (__VLS_ctx.bodyClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_79));
    const { default: __VLS_83 } = __VLS_81.slots;
    if (__VLS_ctx.tableRows.length) {
        for (const [row, rowIndex] of __VLS_vFor((__VLS_ctx.tableRows))) {
            let __VLS_84;
            /** @ts-ignore @type { | typeof __VLS_components.TableRow | typeof __VLS_components.TableRow} */
            TableRow;
            // @ts-ignore
            const __VLS_85 = __VLS_asFunctionalComponent1(__VLS_84, new __VLS_84({
                ...{ 'onClick': {} },
                key: (row.id),
                ...{ class: (__VLS_ctx.getRowClass(row, rowIndex)) },
                dataState: (row.getIsSelected() ? 'selected' : undefined),
            }));
            const __VLS_86 = __VLS_85({
                ...{ 'onClick': {} },
                key: (row.id),
                ...{ class: (__VLS_ctx.getRowClass(row, rowIndex)) },
                dataState: (row.getIsSelected() ? 'selected' : undefined),
            }, ...__VLS_functionalComponentArgsRest(__VLS_85));
            let __VLS_89;
            const __VLS_90 = {
                /** @type {typeof __VLS_89.click} */
                onClick: (...[$event]) => {
                    if (!(__VLS_ctx.tableRows.length))
                        throw 0;
                    return (__VLS_ctx.handleRowClick(row, $event));
                    // @ts-ignore
                    [tableRows, tableRows, bodyClass, getRowClass, handleRowClick,];
                },
            };
            const { default: __VLS_91 } = __VLS_87.slots;
            for (const [cell] of __VLS_vFor((row.getVisibleCells()))) {
                let __VLS_92;
                /** @ts-ignore @type { | typeof __VLS_components.TableCell | typeof __VLS_components.TableCell} */
                TableCell;
                // @ts-ignore
                const __VLS_93 = __VLS_asFunctionalComponent1(__VLS_92, new __VLS_92({
                    key: (cell.id),
                    ...{ class: (__VLS_ctx.getCellClass(cell)) },
                    ...{ style: (__VLS_ctx.getColumnStyle(cell.column)) },
                    dataPinned: (cell.column.getIsPinned() || undefined),
                }));
                const __VLS_94 = __VLS_93({
                    key: (cell.id),
                    ...{ class: (__VLS_ctx.getCellClass(cell)) },
                    ...{ style: (__VLS_ctx.getColumnStyle(cell.column)) },
                    dataPinned: (cell.column.getIsPinned() || undefined),
                }, ...__VLS_functionalComponentArgsRest(__VLS_93));
                const { default: __VLS_97 } = __VLS_95.slots;
                if (__VLS_ctx.shouldRenderTreeCell(cell, row)) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                        ...{ class: "flex gap-1.5 min-w-0 items-center" },
                        ...{ style: (__VLS_ctx.getTreeCellIndentStyle(row)) },
                    });
                    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
                    /** @type {__VLS_StyleScopedClasses['gap-1.5']} */ ;
                    /** @type {__VLS_StyleScopedClasses['min-w-0']} */ ;
                    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
                    if (row.getCanExpand()) {
                        const __VLS_98 = Button || Button;
                        // @ts-ignore
                        const __VLS_99 = __VLS_asFunctionalComponent1(__VLS_98, new __VLS_98({
                            ...{ 'onClick': {} },
                            type: "button",
                            variant: "ghost",
                            size: "icon",
                            ...{ class: "shrink-0 size-6" },
                        }));
                        const __VLS_100 = __VLS_99({
                            ...{ 'onClick': {} },
                            type: "button",
                            variant: "ghost",
                            size: "icon",
                            ...{ class: "shrink-0 size-6" },
                        }, ...__VLS_functionalComponentArgsRest(__VLS_99));
                        let __VLS_103;
                        const __VLS_104 = {
                            /** @type {typeof __VLS_103.click} */
                            onClick: (...[$event]) => {
                                if (!(__VLS_ctx.tableRows.length))
                                    throw 0;
                                if (!(__VLS_ctx.shouldRenderTreeCell(cell, row)))
                                    throw 0;
                                if (!(row.getCanExpand()))
                                    throw 0;
                                return (__VLS_ctx.handleToggleRowExpanded(row));
                                // @ts-ignore
                                [getCellClass, getColumnStyle, shouldRenderTreeCell, getTreeCellIndentStyle, handleToggleRowExpanded,];
                            },
                        };
                        /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
                        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
                        const { default: __VLS_105 } = __VLS_101.slots;
                        const __VLS_106 = Icon;
                        // @ts-ignore
                        const __VLS_107 = __VLS_asFunctionalComponent1(__VLS_106, new __VLS_106({
                            name: (__VLS_ctx.getTreeToggleIconName(row)),
                            ...{ class: "size-3.5" },
                        }));
                        const __VLS_108 = __VLS_107({
                            name: (__VLS_ctx.getTreeToggleIconName(row)),
                            ...{ class: "size-3.5" },
                        }, ...__VLS_functionalComponentArgsRest(__VLS_107));
                        /** @type {__VLS_StyleScopedClasses['size-3.5']} */ ;
                        // @ts-ignore
                        [getTreeToggleIconName,];
                        var __VLS_101;
                        var __VLS_102;
                    }
                    else {
                        __VLS_asFunctionalElement1(__VLS_intrinsics.span)({
                            ...{ class: "shrink-0 size-6" },
                            'aria-hidden': "true",
                        });
                        /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
                        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
                    }
                    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                        ...{ class: "flex-1 min-w-0" },
                    });
                    /** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
                    /** @type {__VLS_StyleScopedClasses['min-w-0']} */ ;
                    if (__VLS_ctx.hasColumnSlot(__VLS_ctx.getCellSlotName(cell))) {
                        __VLS_asFunctionalSlot(slots[(__VLS_ctx.getCellSlotName(cell))])({
                            ...(__VLS_ctx.getCellSlotProps(cell, row)),
                        });
                    }
                    else if (__VLS_ctx.isSelectionColumn(cell.column) && __VLS_ctx.selectable) {
                        if (__VLS_ctx.multiple) {
                            let __VLS_112;
                            /** @ts-ignore @type { | typeof __VLS_components.Checkbox} */
                            Checkbox;
                            // @ts-ignore
                            const __VLS_113 = __VLS_asFunctionalComponent1(__VLS_112, new __VLS_112({
                                ...{ 'onClick': {} },
                                ...{ 'onUpdate:modelValue': {} },
                                modelValue: (row.getIsSelected()),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }));
                            const __VLS_114 = __VLS_113({
                                ...{ 'onClick': {} },
                                ...{ 'onUpdate:modelValue': {} },
                                modelValue: (row.getIsSelected()),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }, ...__VLS_functionalComponentArgsRest(__VLS_113));
                            let __VLS_117;
                            const __VLS_118 = {
                                /** @type {typeof __VLS_117.click} */
                                onClick: () => { },
                            };
                            const __VLS_119 = {
                                /** @type {typeof __VLS_117.'update:modelValue'} */
                                'onUpdate:modelValue': (value => __VLS_ctx.handleToggleRow(row, value)),
                            };
                            /** @type {__VLS_StyleScopedClasses['translate-y-[2px]']} */ ;
                            var __VLS_115;
                            var __VLS_116;
                        }
                        else {
                            let __VLS_120;
                            /** @ts-ignore @type { | typeof __VLS_components.RadioGroupItem} */
                            RadioGroupItem;
                            // @ts-ignore
                            const __VLS_121 = __VLS_asFunctionalComponent1(__VLS_120, new __VLS_120({
                                ...{ 'onClick': {} },
                                value: (row.id),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }));
                            const __VLS_122 = __VLS_121({
                                ...{ 'onClick': {} },
                                value: (row.id),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }, ...__VLS_functionalComponentArgsRest(__VLS_121));
                            let __VLS_125;
                            const __VLS_126 = {
                                /** @type {typeof __VLS_125.click} */
                                onClick: () => { },
                            };
                            /** @type {__VLS_StyleScopedClasses['translate-y-[2px]']} */ ;
                            var __VLS_123;
                            var __VLS_124;
                        }
                    }
                    else {
                        let __VLS_127;
                        /** @ts-ignore @type { | typeof __VLS_components.FlexRender} */
                        FlexRender;
                        // @ts-ignore
                        const __VLS_128 = __VLS_asFunctionalComponent1(__VLS_127, new __VLS_127({
                            cell: (cell),
                        }));
                        const __VLS_129 = __VLS_128({
                            cell: (cell),
                        }, ...__VLS_functionalComponentArgsRest(__VLS_128));
                    }
                }
                else {
                    if (__VLS_ctx.hasColumnSlot(__VLS_ctx.getCellSlotName(cell))) {
                        __VLS_asFunctionalSlot(slots[(__VLS_ctx.getCellSlotName(cell))])({
                            ...(__VLS_ctx.getCellSlotProps(cell, row)),
                        });
                    }
                    else if (__VLS_ctx.isSelectionColumn(cell.column) && __VLS_ctx.selectable) {
                        if (__VLS_ctx.multiple) {
                            let __VLS_133;
                            /** @ts-ignore @type { | typeof __VLS_components.Checkbox} */
                            Checkbox;
                            // @ts-ignore
                            const __VLS_134 = __VLS_asFunctionalComponent1(__VLS_133, new __VLS_133({
                                ...{ 'onClick': {} },
                                ...{ 'onUpdate:modelValue': {} },
                                modelValue: (row.getIsSelected()),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }));
                            const __VLS_135 = __VLS_134({
                                ...{ 'onClick': {} },
                                ...{ 'onUpdate:modelValue': {} },
                                modelValue: (row.getIsSelected()),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }, ...__VLS_functionalComponentArgsRest(__VLS_134));
                            let __VLS_138;
                            const __VLS_139 = {
                                /** @type {typeof __VLS_138.click} */
                                onClick: () => { },
                            };
                            const __VLS_140 = {
                                /** @type {typeof __VLS_138.'update:modelValue'} */
                                'onUpdate:modelValue': (value => __VLS_ctx.handleToggleRow(row, value)),
                            };
                            /** @type {__VLS_StyleScopedClasses['translate-y-[2px]']} */ ;
                            var __VLS_136;
                            var __VLS_137;
                        }
                        else {
                            let __VLS_141;
                            /** @ts-ignore @type { | typeof __VLS_components.RadioGroupItem} */
                            RadioGroupItem;
                            // @ts-ignore
                            const __VLS_142 = __VLS_asFunctionalComponent1(__VLS_141, new __VLS_141({
                                ...{ 'onClick': {} },
                                value: (row.id),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }));
                            const __VLS_143 = __VLS_142({
                                ...{ 'onClick': {} },
                                value: (row.id),
                                disabled: (!row.getCanSelect()),
                                ...{ class: "translate-y-[2px]" },
                            }, ...__VLS_functionalComponentArgsRest(__VLS_142));
                            let __VLS_146;
                            const __VLS_147 = {
                                /** @type {typeof __VLS_146.click} */
                                onClick: () => { },
                            };
                            /** @type {__VLS_StyleScopedClasses['translate-y-[2px]']} */ ;
                            var __VLS_144;
                            var __VLS_145;
                        }
                    }
                    else {
                        let __VLS_148;
                        /** @ts-ignore @type { | typeof __VLS_components.FlexRender} */
                        FlexRender;
                        // @ts-ignore
                        const __VLS_149 = __VLS_asFunctionalComponent1(__VLS_148, new __VLS_148({
                            cell: (cell),
                        }));
                        const __VLS_150 = __VLS_149({
                            cell: (cell),
                        }, ...__VLS_functionalComponentArgsRest(__VLS_149));
                    }
                }
                // @ts-ignore
                [hasColumnSlot, hasColumnSlot, getCellSlotName, getCellSlotName, getCellSlotName, getCellSlotName, getCellSlotProps, getCellSlotProps, isSelectionColumn, isSelectionColumn, selectable, selectable, multiple, multiple, handleToggleRow, handleToggleRow,];
                var __VLS_95;
                // @ts-ignore
                [];
            }
            // @ts-ignore
            [];
            var __VLS_87;
            var __VLS_88;
            // @ts-ignore
            [];
        }
    }
    else {
        let __VLS_153;
        /** @ts-ignore @type { | typeof __VLS_components.TableEmpty | typeof __VLS_components.TableEmpty} */
        TableEmpty;
        // @ts-ignore
        const __VLS_154 = __VLS_asFunctionalComponent1(__VLS_153, new __VLS_153({
            colspan: (__VLS_ctx.visibleColumnCount),
        }));
        const __VLS_155 = __VLS_154({
            colspan: (__VLS_ctx.visibleColumnCount),
        }, ...__VLS_functionalComponentArgsRest(__VLS_154));
        const { default: __VLS_158 } = __VLS_156.slots;
        __VLS_asFunctionalSlot(slots.empty)({
            table: (__VLS_ctx.table),
        });
        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
            ...{ class: "text-muted-foreground" },
        });
        /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
        (__VLS_ctx.emptyText);
        // @ts-ignore
        [table, visibleColumnCount, emptyText,];
        var __VLS_156;
    }
    // @ts-ignore
    [];
    var __VLS_81;
    // @ts-ignore
    [];
    var __VLS_29;
    // @ts-ignore
    [];
    var __VLS_21;
    var __VLS_22;
    // @ts-ignore
    [];
    return {};
})()) => ({}));
