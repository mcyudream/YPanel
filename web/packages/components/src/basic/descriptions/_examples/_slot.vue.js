// 组件实际使用时无需手动导入，框架会自动导入
import FaIcon from '../../icon/index.vue';
import FaTag from '../../tag/index.vue';
import FaDescriptions from '../index.vue';
const items = [
    { key: 'username', label: 'Username', value: 'kooriokami', icon: 'i-lucide:user' },
    { key: 'telephone', label: 'Telephone', value: '18100000000', icon: 'i-lucide:smartphone' },
    { key: 'place', label: 'Place', value: 'Suzhou', icon: 'i-lucide:map-pin' },
    { key: 'remarks', label: 'Remarks', value: 'School', icon: 'i-lucide:notebook-text', tagVariant: 'outline' },
    { key: 'address', label: 'Address', value: 'No.1188, Wuzhong Avenue, Wuzhong District, Suzhou, Jiangsu Province', icon: 'i-lucide:building-2', span: 2 },
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaDescriptions || FaDescriptions;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    items: (__VLS_ctx.items),
    border: true,
    labelWidth: "200px",
}));
const __VLS_2 = __VLS_1({
    items: (__VLS_ctx.items),
    border: true,
    labelWidth: "200px",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { 'label-username': __VLS_7 } = __VLS_3.slots;
    const [{ item, label }] = __VLS_vSlot(__VLS_7);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "inline-flex gap-2 items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    const __VLS_8 = FaIcon;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        name: (item.icon),
        ...{ class: "size-4" },
    }));
    const __VLS_10 = __VLS_9({
        name: (item.icon),
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    (label);
    // @ts-ignore
    [items,];
}
{
    const { 'label-telephone': __VLS_13 } = __VLS_3.slots;
    const [{ item, label }] = __VLS_vSlot(__VLS_13);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "inline-flex gap-2 items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    const __VLS_14 = FaIcon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        name: (item.icon),
        ...{ class: "size-4" },
    }));
    const __VLS_16 = __VLS_15({
        name: (item.icon),
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    (label);
    // @ts-ignore
    [];
}
{
    const { 'label-place': __VLS_19 } = __VLS_3.slots;
    const [{ item, label }] = __VLS_vSlot(__VLS_19);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "inline-flex gap-2 items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    const __VLS_20 = FaIcon;
    // @ts-ignore
    const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({
        name: (item.icon),
        ...{ class: "size-4" },
    }));
    const __VLS_22 = __VLS_21({
        name: (item.icon),
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_21));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    (label);
    // @ts-ignore
    [];
}
{
    const { 'label-remarks': __VLS_25 } = __VLS_3.slots;
    const [{ item, label }] = __VLS_vSlot(__VLS_25);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "inline-flex gap-2 items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    const __VLS_26 = FaIcon;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        name: (item.icon),
        ...{ class: "size-4" },
    }));
    const __VLS_28 = __VLS_27({
        name: (item.icon),
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    (label);
    // @ts-ignore
    [];
}
{
    const { 'label-address': __VLS_31 } = __VLS_3.slots;
    const [{ item, label }] = __VLS_vSlot(__VLS_31);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "inline-flex gap-2 items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    const __VLS_32 = FaIcon;
    // @ts-ignore
    const __VLS_33 = __VLS_asFunctionalComponent1(__VLS_32, new __VLS_32({
        name: (item.icon),
        ...{ class: "size-4" },
    }));
    const __VLS_34 = __VLS_33({
        name: (item.icon),
        ...{ class: "size-4" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_33));
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
    (label);
    // @ts-ignore
    [];
}
{
    const { 'value-remarks': __VLS_37 } = __VLS_3.slots;
    const [{ item, value }] = __VLS_vSlot(__VLS_37);
    const __VLS_38 = FaTag || FaTag;
    // @ts-ignore
    const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({
        variant: (item.tagVariant),
    }));
    const __VLS_40 = __VLS_39({
        variant: (item.tagVariant),
    }, ...__VLS_functionalComponentArgsRest(__VLS_39));
    const { default: __VLS_43 } = __VLS_41.slots;
    (value);
    // @ts-ignore
    [];
    var __VLS_41;
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
