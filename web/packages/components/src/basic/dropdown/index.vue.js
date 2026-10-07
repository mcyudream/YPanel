import { createReusableTemplate, useTextDirection } from '@vueuse/core';
import Icon from '../icon/index.vue';
import { DropdownMenu, DropdownMenuCheckboxItem, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuSub, DropdownMenuSubContent, DropdownMenuSubTrigger, DropdownMenuTrigger, } from './dropdown-menu';
defineOptions({
    name: 'BuiltInDropdown',
});
const __VLS_props = defineProps();
const slots = defineSlots();
const Option = createReusableTemplate();
const dir = useTextDirection({
    observe: true,
});
function hasIcon(group) {
    return group.some(item => item.some((v) => {
        return 'icon' in v && v.icon;
    }));
}
function handleItemClick(item) {
    item.handle?.();
}
function handleCheckboxItemChange(item, checked) {
    item.handle?.(checked);
}
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.DropdownMenu | typeof __VLS_components.DropdownMenu} */
DropdownMenu;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modal: (false),
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
}));
const __VLS_2 = __VLS_1({
    modal: (false),
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.DropdownMenuTrigger | typeof __VLS_components.DropdownMenuTrigger} */
DropdownMenuTrigger;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    asChild: true,
}));
const __VLS_9 = __VLS_8({
    asChild: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
__VLS_asFunctionalSlot(slots['default'])({});
// @ts-ignore
[dir,];
var __VLS_10;
let __VLS_14;
/** @ts-ignore @type { | typeof __VLS_components.DropdownMenuContent | typeof __VLS_components.DropdownMenuContent} */
DropdownMenuContent;
// @ts-ignore
const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
    align: __VLS_ctx.align,
    alignOffset: __VLS_ctx.alignOffset,
    side: __VLS_ctx.side,
    sideOffset: __VLS_ctx.sideOffset,
    collisionPadding: __VLS_ctx.collisionPadding,
    ...{ class: "z-2000" },
}));
const __VLS_16 = __VLS_15({
    align: __VLS_ctx.align,
    alignOffset: __VLS_ctx.alignOffset,
    side: __VLS_ctx.side,
    sideOffset: __VLS_ctx.sideOffset,
    collisionPadding: __VLS_ctx.collisionPadding,
    ...{ class: "z-2000" },
}, ...__VLS_functionalComponentArgsRest(__VLS_15));
/** @type {__VLS_StyleScopedClasses['z-2000']} */ ;
const { default: __VLS_19 } = __VLS_17.slots;
if (!!slots.header) {
    let __VLS_20;
    /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuLabel | typeof __VLS_components.DropdownMenuLabel} */
    DropdownMenuLabel;
    // @ts-ignore
    const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({}));
    const __VLS_22 = __VLS_21({}, ...__VLS_functionalComponentArgsRest(__VLS_21));
    const { default: __VLS_25 } = __VLS_23.slots;
    __VLS_asFunctionalSlot(slots.header)({});
    // @ts-ignore
    [align, alignOffset, side, sideOffset, collisionPadding,];
    var __VLS_23;
    let __VLS_27;
    /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuSeparator} */
    DropdownMenuSeparator;
    // @ts-ignore
    const __VLS_28 = __VLS_asFunctionalComponent1(__VLS_27, new __VLS_27({}));
    const __VLS_29 = __VLS_28({}, ...__VLS_functionalComponentArgsRest(__VLS_28));
}
const __VLS_32 = (__VLS_ctx.Option.define) || (__VLS_ctx.Option.define);
// @ts-ignore
const __VLS_33 = __VLS_asFunctionalComponent1(__VLS_32, new __VLS_32({}));
const __VLS_34 = __VLS_33({}, ...__VLS_functionalComponentArgsRest(__VLS_33));
{
    const { default: __VLS_37 } = __VLS_35.slots;
    const [{ items: its }] = __VLS_vSlot(__VLS_37);
    for (const [item, index] of __VLS_vFor((its))) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
            key: (index),
        });
        let __VLS_38;
        /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuGroup | typeof __VLS_components.DropdownMenuGroup} */
        DropdownMenuGroup;
        // @ts-ignore
        const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({}));
        const __VLS_40 = __VLS_39({}, ...__VLS_functionalComponentArgsRest(__VLS_39));
        const { default: __VLS_43 } = __VLS_41.slots;
        for (const [v, i] of __VLS_vFor((item))) {
            __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
                key: (i),
            });
            if (!('items' in v) && v.type === 'checkbox') {
                let __VLS_44;
                /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuCheckboxItem | typeof __VLS_components.DropdownMenuCheckboxItem} */
                DropdownMenuCheckboxItem;
                // @ts-ignore
                const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
                    ...{ 'onUpdate:modelValue': {} },
                    modelValue: (v.checked),
                    disabled: (v.disabled),
                }));
                const __VLS_46 = __VLS_45({
                    ...{ 'onUpdate:modelValue': {} },
                    modelValue: (v.checked),
                    disabled: (v.disabled),
                }, ...__VLS_functionalComponentArgsRest(__VLS_45));
                let __VLS_49;
                const __VLS_50 = {
                    /** @type {typeof __VLS_49.'update:modelValue'} */
                    'onUpdate:modelValue': (value => __VLS_ctx.handleCheckboxItemChange(v, value)),
                };
                const { default: __VLS_51 } = __VLS_47.slots;
                (v.label);
                // @ts-ignore
                [Option, Option, handleCheckboxItemChange,];
                var __VLS_47;
                var __VLS_48;
            }
            else if (!('items' in v)) {
                let __VLS_52;
                /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuItem | typeof __VLS_components.DropdownMenuItem} */
                DropdownMenuItem;
                // @ts-ignore
                const __VLS_53 = __VLS_asFunctionalComponent1(__VLS_52, new __VLS_52({
                    ...{ 'onClick': {} },
                    variant: (v.variant),
                    disabled: (v.disabled),
                }));
                const __VLS_54 = __VLS_53({
                    ...{ 'onClick': {} },
                    variant: (v.variant),
                    disabled: (v.disabled),
                }, ...__VLS_functionalComponentArgsRest(__VLS_53));
                let __VLS_57;
                const __VLS_58 = {
                    /** @type {typeof __VLS_57.click} */
                    onClick: (...[$event]) => {
                        if (!!(!('items' in v) && v.type === 'checkbox'))
                            throw 0;
                        if (!(!('items' in v)))
                            throw 0;
                        return (__VLS_ctx.handleItemClick(v));
                        // @ts-ignore
                        [handleItemClick,];
                    },
                };
                const { default: __VLS_59 } = __VLS_55.slots;
                if (__VLS_ctx.hasIcon(its)) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                        ...{ class: "flex-center size-4" },
                    });
                    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
                    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
                    if (v.icon) {
                        const __VLS_60 = Icon;
                        // @ts-ignore
                        const __VLS_61 = __VLS_asFunctionalComponent1(__VLS_60, new __VLS_60({
                            name: (v.icon),
                            ...{ class: "size-4" },
                        }));
                        const __VLS_62 = __VLS_61({
                            name: (v.icon),
                            ...{ class: "size-4" },
                        }, ...__VLS_functionalComponentArgsRest(__VLS_61));
                        /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
                    }
                }
                (v.label);
                // @ts-ignore
                [hasIcon,];
                var __VLS_55;
                var __VLS_56;
            }
            else {
                let __VLS_65;
                /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuSub | typeof __VLS_components.DropdownMenuSub} */
                DropdownMenuSub;
                // @ts-ignore
                const __VLS_66 = __VLS_asFunctionalComponent1(__VLS_65, new __VLS_65({}));
                const __VLS_67 = __VLS_66({}, ...__VLS_functionalComponentArgsRest(__VLS_66));
                const { default: __VLS_70 } = __VLS_68.slots;
                let __VLS_71;
                /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuSubTrigger | typeof __VLS_components.DropdownMenuSubTrigger} */
                DropdownMenuSubTrigger;
                // @ts-ignore
                const __VLS_72 = __VLS_asFunctionalComponent1(__VLS_71, new __VLS_71({
                    inset: (__VLS_ctx.hasIcon(its)),
                }));
                const __VLS_73 = __VLS_72({
                    inset: (__VLS_ctx.hasIcon(its)),
                }, ...__VLS_functionalComponentArgsRest(__VLS_72));
                const { default: __VLS_76 } = __VLS_74.slots;
                (v.label);
                // @ts-ignore
                [hasIcon,];
                var __VLS_74;
                let __VLS_77;
                /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuSubContent | typeof __VLS_components.DropdownMenuSubContent} */
                DropdownMenuSubContent;
                // @ts-ignore
                const __VLS_78 = __VLS_asFunctionalComponent1(__VLS_77, new __VLS_77({}));
                const __VLS_79 = __VLS_78({}, ...__VLS_functionalComponentArgsRest(__VLS_78));
                const { default: __VLS_82 } = __VLS_80.slots;
                const __VLS_83 = (__VLS_ctx.Option.reuse);
                // @ts-ignore
                const __VLS_84 = __VLS_asFunctionalComponent1(__VLS_83, new __VLS_83({
                    items: (v.items),
                }));
                const __VLS_85 = __VLS_84({
                    items: (v.items),
                }, ...__VLS_functionalComponentArgsRest(__VLS_84));
                // @ts-ignore
                [Option,];
                var __VLS_80;
                // @ts-ignore
                [];
                var __VLS_68;
            }
            // @ts-ignore
            [];
        }
        // @ts-ignore
        [];
        var __VLS_41;
        if (item.length > 0 && index !== its.length - 1) {
            let __VLS_88;
            /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuSeparator} */
            DropdownMenuSeparator;
            // @ts-ignore
            const __VLS_89 = __VLS_asFunctionalComponent1(__VLS_88, new __VLS_88({}));
            const __VLS_90 = __VLS_89({}, ...__VLS_functionalComponentArgsRest(__VLS_89));
        }
        // @ts-ignore
        [];
    }
    // @ts-ignore
    [];
    __VLS_35.slots['' /* empty slot name completion */];
}
var __VLS_35;
const __VLS_93 = (__VLS_ctx.Option.reuse);
// @ts-ignore
const __VLS_94 = __VLS_asFunctionalComponent1(__VLS_93, new __VLS_93({
    items: (__VLS_ctx.items),
}));
const __VLS_95 = __VLS_94({
    items: (__VLS_ctx.items),
}, ...__VLS_functionalComponentArgsRest(__VLS_94));
if (!!slots.footer) {
    let __VLS_98;
    /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuSeparator} */
    DropdownMenuSeparator;
    // @ts-ignore
    const __VLS_99 = __VLS_asFunctionalComponent1(__VLS_98, new __VLS_98({}));
    const __VLS_100 = __VLS_99({}, ...__VLS_functionalComponentArgsRest(__VLS_99));
    let __VLS_103;
    /** @ts-ignore @type { | typeof __VLS_components.DropdownMenuLabel | typeof __VLS_components.DropdownMenuLabel} */
    DropdownMenuLabel;
    // @ts-ignore
    const __VLS_104 = __VLS_asFunctionalComponent1(__VLS_103, new __VLS_103({}));
    const __VLS_105 = __VLS_104({}, ...__VLS_functionalComponentArgsRest(__VLS_104));
    const { default: __VLS_108 } = __VLS_106.slots;
    __VLS_asFunctionalSlot(slots.footer)({});
    // @ts-ignore
    [Option, items,];
    var __VLS_106;
}
// @ts-ignore
[];
var __VLS_17;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
