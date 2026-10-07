import { computed } from 'vue';
import { cn } from '#utils';
export default {};
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInDescriptions',
    });
    const props = withDefaults(defineProps(), {
        column: 3,
        direction: 'horizontal',
        border: false,
        size: 'default',
        emptyText: '-',
    });
    const __VLS_slots = defineSlots();
    const cellSizeClasses = {
        sm: 'px-3 py-2 text-xs',
        default: 'px-4 py-3 text-sm',
        lg: 'px-5 py-4 text-base',
    };
    const normalizedColumn = computed(() => normalizeColumn(props.column));
    const rows = computed(() => createRows(props.items ?? [], normalizedColumn.value));
    const rootClass = computed(() => cn('w-full overflow-hidden', props.border && 'rounded-lg border', props.class));
    const tableClass = computed(() => cn('w-full table-fixed border-collapse', !props.border && 'border-separate border-spacing-0'));
    const labelColStyle = computed(() => {
        if (props.direction !== 'horizontal') {
            return undefined;
        }
        const width = resolveSize(props.labelWidth);
        return width ? { width } : undefined;
    });
    function normalizeColumn(column) {
        const resolved = Number(column);
        return Number.isFinite(resolved) && resolved > 0 ? Math.floor(resolved) : 1;
    }
    function normalizeSpan(span, column) {
        const resolved = Number(span);
        if (!Number.isFinite(resolved) || resolved <= 0) {
            return 1;
        }
        return Math.min(Math.floor(resolved), column);
    }
    function createRows(items, column) {
        const rows = [];
        let currentItems = [];
        let currentSpan = 0;
        function pushCurrentRow() {
            if (!currentItems.length) {
                return;
            }
            rows.push({
                items: currentItems,
                span: currentSpan,
                rest: Math.max(column - currentSpan, 0),
            });
            currentItems = [];
            currentSpan = 0;
        }
        items.forEach((item, index) => {
            const span = normalizeSpan(item.span, column);
            if (currentItems.length && currentSpan + span > column) {
                pushCurrentRow();
            }
            currentItems.push({
                item,
                index,
                span,
            });
            currentSpan += span;
            if (currentSpan >= column) {
                pushCurrentRow();
            }
        });
        pushCurrentRow();
        return rows;
    }
    function resolveSize(size) {
        if (size == null || size === '') {
            return undefined;
        }
        return typeof size === 'number' ? `${size}px` : size;
    }
    function isEmptyValue(value) {
        return value === null || value === undefined || value === '';
    }
    function resolveValue(value) {
        return isEmptyValue(value) ? props.emptyText : value;
    }
    function getLabelSlotName(item) {
        return item.key ? `label-${item.key}` : undefined;
    }
    function getValueSlotName(item) {
        return item.key ? `value-${item.key}` : undefined;
    }
    function getRowClass(isLastRow) {
        return cn(props.border && !isLastRow && 'border-b');
    }
    function getBaseCellClass(isLastCell) {
        return cn('align-middle break-words', cellSizeClasses[props.size], props.border && !isLastCell && 'border-r');
    }
    function getLabelCellClass(entry, isLastCell) {
        return cn(getBaseCellClass(isLastCell), 'text-left font-medium text-muted-foreground', props.border && 'bg-muted/50', entry.item.class, props.labelClass, entry.item.labelClass);
    }
    function getValueCellClass(entry, isLastCell) {
        return cn(getBaseCellClass(isLastCell), 'text-foreground', entry.item.class, props.valueClass, entry.item.valueClass);
    }
    function getPlaceholderCellClass(isLastCell) {
        return cn(getBaseCellClass(isLastCell), props.border && 'bg-background');
    }
    function isLastContentCell(row, index) {
        return index === row.items.length - 1 && row.rest === 0;
    }
    const __VLS_defaults = {
        column: 3,
        direction: 'horizontal',
        border: false,
        size: 'default',
        emptyText: '-',
    };
    const __VLS_ctx = {
        ...{},
        ...{},
        ...{},
        ...{},
    };
    let __VLS_components;
    let __VLS_intrinsics;
    let __VLS_directives;
    if (__VLS_ctx.rows.length) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: (__VLS_ctx.rootClass) },
            'data-slot': "descriptions",
        });
        __VLS_asFunctionalElement1(__VLS_intrinsics.table, __VLS_intrinsics.table)({
            ...{ class: (__VLS_ctx.tableClass) },
            'data-slot': "descriptions-table",
        });
        if (__VLS_ctx.direction === 'horizontal') {
            __VLS_asFunctionalElement1(__VLS_intrinsics.colgroup, __VLS_intrinsics.colgroup)({});
            for (const [columnIndex] of __VLS_vFor((__VLS_ctx.normalizedColumn))) {
                __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
                    key: (columnIndex),
                });
                __VLS_asFunctionalElement1(__VLS_intrinsics.col)({
                    ...{ style: (__VLS_ctx.labelColStyle) },
                });
                __VLS_asFunctionalElement1(__VLS_intrinsics.col)({});
                // @ts-ignore
                [rows, rootClass, tableClass, direction, normalizedColumn, labelColStyle,];
            }
        }
        else {
            __VLS_asFunctionalElement1(__VLS_intrinsics.colgroup, __VLS_intrinsics.colgroup)({});
            for (const [columnIndex] of __VLS_vFor((__VLS_ctx.normalizedColumn))) {
                __VLS_asFunctionalElement1(__VLS_intrinsics.col, __VLS_intrinsics.col)({
                    key: (columnIndex),
                });
                // @ts-ignore
                [normalizedColumn,];
            }
        }
        __VLS_asFunctionalElement1(__VLS_intrinsics.tbody, __VLS_intrinsics.tbody)({});
        if (__VLS_ctx.direction === 'vertical') {
            for (const [row, rowIndex] of __VLS_vFor((__VLS_ctx.rows))) {
                __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
                    key: (rowIndex),
                });
                __VLS_asFunctionalElement1(__VLS_intrinsics.tr, __VLS_intrinsics.tr)({
                    ...{ class: (__VLS_ctx.getRowClass(false)) },
                    'data-slot': "descriptions-label-row",
                });
                for (const [entry, entryIndex] of __VLS_vFor((row.items))) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.th, __VLS_intrinsics.th)({
                        key: (`label-${entry.index}`),
                        colspan: (entry.span),
                        ...{ class: (__VLS_ctx.getLabelCellClass(entry, __VLS_ctx.isLastContentCell(row, entryIndex))) },
                        scope: "col",
                        'data-slot': "descriptions-label",
                    });
                    if (__VLS_ctx.getLabelSlotName(entry.item)) {
                        __VLS_asFunctionalSlot(__VLS_slots[(__VLS_ctx.getLabelSlotName(entry.item))])({
                            item: (entry.item),
                            index: (entry.index),
                            label: (entry.item.label),
                            value: (entry.item.value),
                        });
                        (entry.item.label);
                    }
                    else {
                        (entry.item.label);
                    }
                    // @ts-ignore
                    [rows, direction, getRowClass, getLabelCellClass, isLastContentCell, getLabelSlotName, getLabelSlotName,];
                }
                if (row.rest) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.td)({
                        colspan: (row.rest),
                        ...{ class: (__VLS_ctx.getPlaceholderCellClass(true)) },
                        'aria-hidden': "true",
                        'data-slot': "descriptions-placeholder",
                    });
                }
                __VLS_asFunctionalElement1(__VLS_intrinsics.tr, __VLS_intrinsics.tr)({
                    ...{ class: (__VLS_ctx.getRowClass(rowIndex === __VLS_ctx.rows.length - 1)) },
                    'data-slot': "descriptions-value-row",
                });
                for (const [entry, entryIndex] of __VLS_vFor((row.items))) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.td, __VLS_intrinsics.td)({
                        key: (`value-${entry.index}`),
                        colspan: (entry.span),
                        ...{ class: (__VLS_ctx.getValueCellClass(entry, __VLS_ctx.isLastContentCell(row, entryIndex))) },
                        'data-slot': "descriptions-value",
                    });
                    if (__VLS_ctx.getValueSlotName(entry.item)) {
                        __VLS_asFunctionalSlot(__VLS_slots[(__VLS_ctx.getValueSlotName(entry.item))])({
                            item: (entry.item),
                            index: (entry.index),
                            label: (entry.item.label),
                            value: (entry.item.value),
                        });
                        (__VLS_ctx.resolveValue(entry.item.value));
                    }
                    else {
                        (__VLS_ctx.resolveValue(entry.item.value));
                    }
                    // @ts-ignore
                    [rows, getRowClass, isLastContentCell, getPlaceholderCellClass, getValueCellClass, getValueSlotName, getValueSlotName, resolveValue, resolveValue,];
                }
                if (row.rest) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.td)({
                        colspan: (row.rest),
                        ...{ class: (__VLS_ctx.getPlaceholderCellClass(true)) },
                        'aria-hidden': "true",
                        'data-slot': "descriptions-placeholder",
                    });
                }
                // @ts-ignore
                [getPlaceholderCellClass,];
            }
        }
        else {
            for (const [row, rowIndex] of __VLS_vFor((__VLS_ctx.rows))) {
                __VLS_asFunctionalElement1(__VLS_intrinsics.tr, __VLS_intrinsics.tr)({
                    key: (rowIndex),
                    ...{ class: (__VLS_ctx.getRowClass(rowIndex === __VLS_ctx.rows.length - 1)) },
                    'data-slot': "descriptions-row",
                });
                for (const [entry, entryIndex] of __VLS_vFor((row.items))) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
                        key: (entry.index),
                    });
                    __VLS_asFunctionalElement1(__VLS_intrinsics.th, __VLS_intrinsics.th)({
                        ...{ class: (__VLS_ctx.getLabelCellClass(entry, false)) },
                        scope: "row",
                        'data-slot': "descriptions-label",
                    });
                    if (__VLS_ctx.getLabelSlotName(entry.item)) {
                        __VLS_asFunctionalSlot(__VLS_slots[(__VLS_ctx.getLabelSlotName(entry.item))])({
                            item: (entry.item),
                            index: (entry.index),
                            label: (entry.item.label),
                            value: (entry.item.value),
                        });
                        (entry.item.label);
                    }
                    else {
                        (entry.item.label);
                    }
                    __VLS_asFunctionalElement1(__VLS_intrinsics.td, __VLS_intrinsics.td)({
                        colspan: (entry.span * 2 - 1),
                        ...{ class: (__VLS_ctx.getValueCellClass(entry, __VLS_ctx.isLastContentCell(row, entryIndex))) },
                        'data-slot': "descriptions-value",
                    });
                    if (__VLS_ctx.getValueSlotName(entry.item)) {
                        __VLS_asFunctionalSlot(__VLS_slots[(__VLS_ctx.getValueSlotName(entry.item))])({
                            item: (entry.item),
                            index: (entry.index),
                            label: (entry.item.label),
                            value: (entry.item.value),
                        });
                        (__VLS_ctx.resolveValue(entry.item.value));
                    }
                    else {
                        (__VLS_ctx.resolveValue(entry.item.value));
                    }
                    // @ts-ignore
                    [rows, rows, getRowClass, getLabelCellClass, isLastContentCell, getLabelSlotName, getLabelSlotName, getValueCellClass, getValueSlotName, getValueSlotName, resolveValue, resolveValue,];
                }
                if (row.rest) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.td)({
                        colspan: (row.rest * 2),
                        ...{ class: (__VLS_ctx.getPlaceholderCellClass(true)) },
                        'aria-hidden': "true",
                        'data-slot': "descriptions-placeholder",
                    });
                }
                // @ts-ignore
                [getPlaceholderCellClass,];
            }
        }
    }
    // @ts-ignore
    [];
    return {};
})()) => ({}));
