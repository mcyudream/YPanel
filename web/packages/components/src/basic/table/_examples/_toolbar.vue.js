import FaButton from '../../../basic/button/index.vue';
import FaIcon from '../../../basic/icon/index.vue';
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
/** @ts-ignore @type { | typeof __VLS_components.FaTable | typeof __VLS_components.FaTable} */
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
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { toolbar: __VLS_7 } = __VLS_3.slots;
    const [{ table }] = __VLS_vSlot(__VLS_7);
    const __VLS_8 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        size: "sm",
    }));
    const __VLS_10 = __VLS_9({
        size: "sm",
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    const { default: __VLS_13 } = __VLS_11.slots;
    const __VLS_14 = FaIcon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        name: "i-lucide:plus",
    }));
    const __VLS_16 = __VLS_15({
        name: "i-lucide:plus",
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    // @ts-ignore
    [columns, data,];
    var __VLS_11;
    const __VLS_19 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
        variant: "outline",
        size: "sm",
    }));
    const __VLS_21 = __VLS_20({
        variant: "outline",
        size: "sm",
    }, ...__VLS_functionalComponentArgsRest(__VLS_20));
    const { default: __VLS_24 } = __VLS_22.slots;
    const __VLS_25 = FaIcon;
    // @ts-ignore
    const __VLS_26 = __VLS_asFunctionalComponent1(__VLS_25, new __VLS_25({
        name: "i-lucide:refresh-cw",
    }));
    const __VLS_27 = __VLS_26({
        name: "i-lucide:refresh-cw",
    }, ...__VLS_functionalComponentArgsRest(__VLS_26));
    // @ts-ignore
    [];
    var __VLS_22;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "text-sm text-muted-foreground" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    (table.getRowModel().rows.length);
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
