import { createReusableTemplate, useTextDirection } from '@vueuse/core';
import Icon from '../icon/index.vue';
import { ContextMenu, ContextMenuContent, ContextMenuGroup, ContextMenuItem, ContextMenuLabel, ContextMenuSeparator, ContextMenuSub, ContextMenuSubContent, ContextMenuSubTrigger, ContextMenuTrigger, } from './context-menu';
defineOptions({
    name: 'BuiltInContextMenu',
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
/** @ts-ignore @type { | typeof __VLS_components.ContextMenu | typeof __VLS_components.ContextMenu} */
ContextMenu;
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
/** @ts-ignore @type { | typeof __VLS_components.ContextMenuTrigger | typeof __VLS_components.ContextMenuTrigger} */
ContextMenuTrigger;
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
/** @ts-ignore @type { | typeof __VLS_components.ContextMenuContent | typeof __VLS_components.ContextMenuContent} */
ContextMenuContent;
// @ts-ignore
const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
    ...{ class: "z-2050" },
}));
const __VLS_16 = __VLS_15({
    ...{ class: "z-2050" },
}, ...__VLS_functionalComponentArgsRest(__VLS_15));
/** @type {__VLS_StyleScopedClasses['z-2050']} */ ;
const { default: __VLS_19 } = __VLS_17.slots;
if (!!slots.label) {
    let __VLS_20;
    /** @ts-ignore @type { | typeof __VLS_components.ContextMenuLabel | typeof __VLS_components.ContextMenuLabel} */
    ContextMenuLabel;
    // @ts-ignore
    const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({}));
    const __VLS_22 = __VLS_21({}, ...__VLS_functionalComponentArgsRest(__VLS_21));
    const { default: __VLS_25 } = __VLS_23.slots;
    __VLS_asFunctionalSlot(slots.label)({});
    // @ts-ignore
    [];
    var __VLS_23;
    let __VLS_27;
    /** @ts-ignore @type { | typeof __VLS_components.ContextMenuSeparator} */
    ContextMenuSeparator;
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
        /** @ts-ignore @type { | typeof __VLS_components.ContextMenuGroup | typeof __VLS_components.ContextMenuGroup} */
        ContextMenuGroup;
        // @ts-ignore
        const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({}));
        const __VLS_40 = __VLS_39({}, ...__VLS_functionalComponentArgsRest(__VLS_39));
        const { default: __VLS_43 } = __VLS_41.slots;
        for (const [v, i] of __VLS_vFor((item))) {
            __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
                key: (i),
            });
            if (!('items' in v)) {
                let __VLS_44;
                /** @ts-ignore @type { | typeof __VLS_components.ContextMenuItem | typeof __VLS_components.ContextMenuItem} */
                ContextMenuItem;
                // @ts-ignore
                const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
                    ...{ 'onClick': {} },
                    variant: (v.variant),
                    disabled: (v.disabled),
                    ...{ class: "cursor-pointer" },
                }));
                const __VLS_46 = __VLS_45({
                    ...{ 'onClick': {} },
                    variant: (v.variant),
                    disabled: (v.disabled),
                    ...{ class: "cursor-pointer" },
                }, ...__VLS_functionalComponentArgsRest(__VLS_45));
                let __VLS_49;
                const __VLS_50 = {
                    /** @type {typeof __VLS_49.click} */
                    onClick: (...[$event]) => {
                        if (!(!('items' in v)))
                            throw 0;
                        return (__VLS_ctx.handleItemClick(v));
                        // @ts-ignore
                        [Option, Option, handleItemClick,];
                    },
                };
                /** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
                const { default: __VLS_51 } = __VLS_47.slots;
                if (__VLS_ctx.hasIcon(its)) {
                    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                        ...{ class: "flex-center size-4" },
                    });
                    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
                    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
                    if (v.icon) {
                        const __VLS_52 = Icon;
                        // @ts-ignore
                        const __VLS_53 = __VLS_asFunctionalComponent1(__VLS_52, new __VLS_52({
                            name: (v.icon),
                            ...{ class: "size-4" },
                        }));
                        const __VLS_54 = __VLS_53({
                            name: (v.icon),
                            ...{ class: "size-4" },
                        }, ...__VLS_functionalComponentArgsRest(__VLS_53));
                        /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
                    }
                }
                (v.label);
                // @ts-ignore
                [hasIcon,];
                var __VLS_47;
                var __VLS_48;
            }
            else {
                let __VLS_57;
                /** @ts-ignore @type { | typeof __VLS_components.ContextMenuSub | typeof __VLS_components.ContextMenuSub} */
                ContextMenuSub;
                // @ts-ignore
                const __VLS_58 = __VLS_asFunctionalComponent1(__VLS_57, new __VLS_57({}));
                const __VLS_59 = __VLS_58({}, ...__VLS_functionalComponentArgsRest(__VLS_58));
                const { default: __VLS_62 } = __VLS_60.slots;
                let __VLS_63;
                /** @ts-ignore @type { | typeof __VLS_components.ContextMenuSubTrigger | typeof __VLS_components.ContextMenuSubTrigger} */
                ContextMenuSubTrigger;
                // @ts-ignore
                const __VLS_64 = __VLS_asFunctionalComponent1(__VLS_63, new __VLS_63({
                    inset: (__VLS_ctx.hasIcon(its)),
                }));
                const __VLS_65 = __VLS_64({
                    inset: (__VLS_ctx.hasIcon(its)),
                }, ...__VLS_functionalComponentArgsRest(__VLS_64));
                const { default: __VLS_68 } = __VLS_66.slots;
                (v.label);
                // @ts-ignore
                [hasIcon,];
                var __VLS_66;
                let __VLS_69;
                /** @ts-ignore @type { | typeof __VLS_components.ContextMenuSubContent | typeof __VLS_components.ContextMenuSubContent} */
                ContextMenuSubContent;
                // @ts-ignore
                const __VLS_70 = __VLS_asFunctionalComponent1(__VLS_69, new __VLS_69({}));
                const __VLS_71 = __VLS_70({}, ...__VLS_functionalComponentArgsRest(__VLS_70));
                const { default: __VLS_74 } = __VLS_72.slots;
                const __VLS_75 = (__VLS_ctx.Option.reuse);
                // @ts-ignore
                const __VLS_76 = __VLS_asFunctionalComponent1(__VLS_75, new __VLS_75({
                    items: (v.items),
                }));
                const __VLS_77 = __VLS_76({
                    items: (v.items),
                }, ...__VLS_functionalComponentArgsRest(__VLS_76));
                // @ts-ignore
                [Option,];
                var __VLS_72;
                // @ts-ignore
                [];
                var __VLS_60;
            }
            // @ts-ignore
            [];
        }
        // @ts-ignore
        [];
        var __VLS_41;
        if (item.length > 0 && index !== its.length - 1) {
            let __VLS_80;
            /** @ts-ignore @type { | typeof __VLS_components.ContextMenuSeparator} */
            ContextMenuSeparator;
            // @ts-ignore
            const __VLS_81 = __VLS_asFunctionalComponent1(__VLS_80, new __VLS_80({}));
            const __VLS_82 = __VLS_81({}, ...__VLS_functionalComponentArgsRest(__VLS_81));
        }
        // @ts-ignore
        [];
    }
    // @ts-ignore
    [];
    __VLS_35.slots['' /* empty slot name completion */];
}
var __VLS_35;
const __VLS_85 = (__VLS_ctx.Option.reuse);
// @ts-ignore
const __VLS_86 = __VLS_asFunctionalComponent1(__VLS_85, new __VLS_85({
    items: (__VLS_ctx.items),
}));
const __VLS_87 = __VLS_86({
    items: (__VLS_ctx.items),
}, ...__VLS_functionalComponentArgsRest(__VLS_86));
// @ts-ignore
[Option, items,];
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
