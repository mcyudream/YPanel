import { computed, ref, watch } from 'vue';
import Input from '../input/index.vue';
import Select from '../select/index.vue';
import { Pagination, PaginationContent, PaginationEllipsis, PaginationFirst, PaginationItem, PaginationLast, PaginationNext, PaginationPrevious, } from './pagination';
defineOptions({
    name: 'BuiltInPagination',
});
const props = withDefaults(defineProps(), {
    sizes: () => [10, 20, 30, 40, 50, 100],
    layout: 'total, sizes, ->, pager, jumper',
    textTemplates: () => ({
        total: (total) => `共 ${total} 条`,
        sizes: (size) => `${size} 条/页`,
        jumper: { before: '前往', after: '页' },
    }),
});
const emits = defineEmits();
const page = defineModel('page', { required: true });
const size = defineModel('size', { required: true });
const jumpPage = ref(page.value);
const totalPages = computed(() => Math.ceil(props.total / size.value));
// Parse layout string to determine order and visibility
const layoutConfig = computed(() => {
    const items = props.layout.split(',').map(item => item.trim());
    const config = {
        'pager': { show: false, order: 0 },
        'total': { show: false, order: 0 },
        'sizes': { show: false, order: 0 },
        'jumper': { show: false, order: 0 },
        '->': { show: false, order: 0 },
    };
    items.forEach((item, index) => {
        if (config[item] !== undefined) {
            config[item] = { show: true, order: index + 1 };
        }
    });
    return config;
});
// Sync jumpPage with page model and emit pageChange event
watch(page, (newPage) => {
    jumpPage.value = newPage;
    emits('pageChange', newPage);
});
// Emit sizeChange event when size changes
watch(size, (newSize) => {
    emits('sizeChange', newSize);
});
function handleFocus(event) {
    const input = event.target;
    input.select();
}
function handleInput(event) {
    const input = event.target;
    const value = input.value.replace(/\D/g, ''); // Remove non-digit characters
    jumpPage.value = value;
}
function handleJump() {
    const pageNum = Number(jumpPage.value);
    if (pageNum && Number.isInteger(pageNum) && pageNum >= 1 && pageNum <= totalPages.value) {
        page.value = pageNum;
    }
    else if (pageNum > totalPages.value) {
        jumpPage.value = totalPages.value;
    }
    else {
        jumpPage.value = page.value;
    }
}
let __VLS_modelEmit;
const __VLS_defaults = {
    sizes: () => [10, 20, 30, 40, 50, 100],
    layout: 'total, sizes, ->, pager, jumper',
    textTemplates: () => ({
        total: (total) => `共 ${total} 条`,
        sizes: (size) => `${size} 条/页`,
        jumper: { before: '前往', after: '页' },
    }),
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Pagination | typeof __VLS_components.Pagination} */
Pagination;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onUpdate:page': {} },
    total: (props.total),
    siblingCount: (1),
    showEdges: true,
    page: (__VLS_ctx.page),
    ...{ class: "gap-4 items-center" },
    itemsPerPage: (__VLS_ctx.size),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onUpdate:page': {} },
    total: (props.total),
    siblingCount: (1),
    showEdges: true,
    page: (__VLS_ctx.page),
    ...{ class: "gap-4 items-center" },
    itemsPerPage: (__VLS_ctx.size),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.'update:page'} */
    'onUpdate:page': ((val) => __VLS_ctx.page = val),
};
var __VLS_7;
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
{
    const { default: __VLS_8 } = __VLS_3.slots;
    const [{ page: currentPage }] = __VLS_vSlot(__VLS_8);
    if (__VLS_ctx.layoutConfig.pager.show) {
        let __VLS_9;
        /** @ts-ignore @type { | typeof __VLS_components.PaginationContent | typeof __VLS_components.PaginationContent} */
        PaginationContent;
        // @ts-ignore
        const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
            ...{ class: "flex-center gap-1" },
            ...{ style: ({ order: __VLS_ctx.layoutConfig.pager.order }) },
        }));
        const __VLS_11 = __VLS_10({
            ...{ class: "flex-center gap-1" },
            ...{ style: ({ order: __VLS_ctx.layoutConfig.pager.order }) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_10));
        /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
        /** @type {__VLS_StyleScopedClasses['gap-1']} */ ;
        {
            const { default: __VLS_14 } = __VLS_12.slots;
            const [{ items }] = __VLS_vSlot(__VLS_14);
            let __VLS_15;
            /** @ts-ignore @type { | typeof __VLS_components.PaginationFirst} */
            PaginationFirst;
            // @ts-ignore
            const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }));
            const __VLS_17 = __VLS_16({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_16));
            /** @type {__VLS_StyleScopedClasses['size-8']} */ ;
            /** @type {__VLS_StyleScopedClasses['rtl:rotate-180']} */ ;
            let __VLS_20;
            /** @ts-ignore @type { | typeof __VLS_components.PaginationPrevious} */
            PaginationPrevious;
            // @ts-ignore
            const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }));
            const __VLS_22 = __VLS_21({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_21));
            /** @type {__VLS_StyleScopedClasses['size-8']} */ ;
            /** @type {__VLS_StyleScopedClasses['rtl:rotate-180']} */ ;
            for (const [item, index] of __VLS_vFor((items))) {
                __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
                    key: (index),
                });
                if (item.type === 'page') {
                    let __VLS_25;
                    /** @ts-ignore @type { | typeof __VLS_components.PaginationItem | typeof __VLS_components.PaginationItem} */
                    PaginationItem;
                    // @ts-ignore
                    const __VLS_26 = __VLS_asFunctionalComponent1(__VLS_25, new __VLS_25({
                        value: (item.value),
                        isActive: (item.value === currentPage),
                        size: "icon-sm",
                        ...{ class: "min-w-8 w-auto" },
                    }));
                    const __VLS_27 = __VLS_26({
                        value: (item.value),
                        isActive: (item.value === currentPage),
                        size: "icon-sm",
                        ...{ class: "min-w-8 w-auto" },
                    }, ...__VLS_functionalComponentArgsRest(__VLS_26));
                    /** @type {__VLS_StyleScopedClasses['min-w-8']} */ ;
                    /** @type {__VLS_StyleScopedClasses['w-auto']} */ ;
                    const { default: __VLS_30 } = __VLS_28.slots;
                    (item.value);
                    // @ts-ignore
                    [page, page, size, layoutConfig, layoutConfig,];
                    var __VLS_28;
                }
                else {
                    let __VLS_31;
                    /** @ts-ignore @type { | typeof __VLS_components.PaginationEllipsis} */
                    PaginationEllipsis;
                    // @ts-ignore
                    const __VLS_32 = __VLS_asFunctionalComponent1(__VLS_31, new __VLS_31({
                        key: (item.type),
                        index: (index),
                    }));
                    const __VLS_33 = __VLS_32({
                        key: (item.type),
                        index: (index),
                    }, ...__VLS_functionalComponentArgsRest(__VLS_32));
                }
                // @ts-ignore
                [];
            }
            let __VLS_36;
            /** @ts-ignore @type { | typeof __VLS_components.PaginationNext} */
            PaginationNext;
            // @ts-ignore
            const __VLS_37 = __VLS_asFunctionalComponent1(__VLS_36, new __VLS_36({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }));
            const __VLS_38 = __VLS_37({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_37));
            /** @type {__VLS_StyleScopedClasses['size-8']} */ ;
            /** @type {__VLS_StyleScopedClasses['rtl:rotate-180']} */ ;
            let __VLS_41;
            /** @ts-ignore @type { | typeof __VLS_components.PaginationLast} */
            PaginationLast;
            // @ts-ignore
            const __VLS_42 = __VLS_asFunctionalComponent1(__VLS_41, new __VLS_41({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }));
            const __VLS_43 = __VLS_42({
                size: "icon-sm",
                ...{ class: "size-8 rtl:rotate-180" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_42));
            /** @type {__VLS_StyleScopedClasses['size-8']} */ ;
            /** @type {__VLS_StyleScopedClasses['rtl:rotate-180']} */ ;
            // @ts-ignore
            [];
            __VLS_12.slots['' /* empty slot name completion */];
        }
        var __VLS_12;
    }
    if (__VLS_ctx.layoutConfig.total.show) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "text-sm text-muted-foreground" },
            ...{ style: ({ order: __VLS_ctx.layoutConfig.total.order }) },
        });
        /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
        (props.textTemplates.total?.(props.total));
    }
    if (__VLS_ctx.layoutConfig.sizes.show) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ style: ({ order: __VLS_ctx.layoutConfig.sizes.order }) },
        });
        const __VLS_46 = Select;
        // @ts-ignore
        const __VLS_47 = __VLS_asFunctionalComponent1(__VLS_46, new __VLS_46({
            modelValue: (__VLS_ctx.size),
            options: (props.sizes.map(size => ({ label: props.textTemplates.sizes?.(size) ?? '', value: size }))),
            ...{ class: "w-auto" },
        }));
        const __VLS_48 = __VLS_47({
            modelValue: (__VLS_ctx.size),
            options: (props.sizes.map(size => ({ label: props.textTemplates.sizes?.(size) ?? '', value: size }))),
            ...{ class: "w-auto" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_47));
        /** @type {__VLS_StyleScopedClasses['w-auto']} */ ;
    }
    if (__VLS_ctx.layoutConfig.jumper.show) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "flex-center gap-2" },
            ...{ style: ({ order: __VLS_ctx.layoutConfig.jumper.order }) },
        });
        /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
        /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
            ...{ class: "text-sm text-muted-foreground" },
        });
        /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
        (props.textTemplates.jumper?.before);
        const __VLS_51 = Input;
        // @ts-ignore
        const __VLS_52 = __VLS_asFunctionalComponent1(__VLS_51, new __VLS_51({
            ...{ 'onFocus': {} },
            ...{ 'onInput': {} },
            ...{ 'onKeyup': {} },
            modelValue: (__VLS_ctx.jumpPage),
            ...{ class: "h-8 w-16" },
            inputClass: "text-center",
        }));
        const __VLS_53 = __VLS_52({
            ...{ 'onFocus': {} },
            ...{ 'onInput': {} },
            ...{ 'onKeyup': {} },
            modelValue: (__VLS_ctx.jumpPage),
            ...{ class: "h-8 w-16" },
            inputClass: "text-center",
        }, ...__VLS_functionalComponentArgsRest(__VLS_52));
        let __VLS_56;
        const __VLS_57 = {
            /** @type {typeof __VLS_56.focus} */
            onFocus: (__VLS_ctx.handleFocus),
        };
        const __VLS_58 = {
            /** @type {typeof __VLS_56.input} */
            onInput: (__VLS_ctx.handleInput),
        };
        const __VLS_59 = {
            /** @type {typeof __VLS_56.keyup} */
            onKeyup: (__VLS_ctx.handleJump),
        };
        /** @type {__VLS_StyleScopedClasses['h-8']} */ ;
        /** @type {__VLS_StyleScopedClasses['w-16']} */ ;
        var __VLS_54;
        var __VLS_55;
        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
            ...{ class: "text-sm text-muted-foreground" },
        });
        /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
        (props.textTemplates.jumper?.after);
    }
    if (__VLS_ctx.layoutConfig['->'].show) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div)({
            ...{ class: "flex-1" },
            ...{ style: ({ order: __VLS_ctx.layoutConfig['->'].order }) },
        });
        /** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
    }
    // @ts-ignore
    [size, layoutConfig, layoutConfig, layoutConfig, layoutConfig, layoutConfig, layoutConfig, layoutConfig, layoutConfig, jumpPage, handleFocus, handleInput, handleJump,];
    __VLS_3.slots['' /* empty slot name completion */];
}
var __VLS_3;
var __VLS_4;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
export default {};
