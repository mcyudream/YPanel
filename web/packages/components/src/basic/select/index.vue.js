import { useTextDirection } from '@vueuse/core';
import { computed, watch } from 'vue';
import { cn } from '#utils';
import { Select, SelectContent, SelectGroup, SelectItem, SelectLabel, SelectTrigger, SelectValue, } from './select';
defineOptions({
    name: 'BuiltInSelect',
});
const props = defineProps();
const emits = defineEmits();
const value = defineModel();
const dir = useTextDirection({
    observe: true,
});
watch(value, (newValue) => {
    emits('change', newValue);
});
const selectedOption = computed({
    get() {
        // 处理普通选项和分组选项
        if (!props.options || props.options.length === 0) {
            return null;
        }
        for (const option of props.options) {
            if (Object.hasOwn(option, 'options')) {
                // 分组选项
                const group = option;
                const found = group.options.find(opt => opt.value === value.value);
                if (found) {
                    return found;
                }
            }
            else {
                // 普通选项
                const single = option;
                if (single.value === value.value) {
                    return single;
                }
            }
        }
        // 如果没有找到匹配项，返回第一个有效选项
        if (props.options.length > 0) {
            const firstOption = props.options[0];
            if (Object.hasOwn(firstOption, 'options')) {
                const group = firstOption;
                return group.options && group.options.length > 0 ? group.options[0] : null;
            }
            else {
                return firstOption;
            }
        }
        return null;
    },
    set(val) {
        value.value = val?.value || null;
    },
});
let __VLS_modelEmit;
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
/** @ts-ignore @type { | typeof __VLS_components.Select | typeof __VLS_components.Select} */
Select;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.value),
    multiple: __VLS_ctx.multiple,
    disabled: __VLS_ctx.disabled,
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.value),
    multiple: __VLS_ctx.multiple,
    disabled: __VLS_ctx.disabled,
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.SelectTrigger | typeof __VLS_components.SelectTrigger} */
SelectTrigger;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    ...{ class: (__VLS_ctx.cn('w-[200px]', props.class)) },
}));
const __VLS_9 = __VLS_8({
    ...{ class: (__VLS_ctx.cn('w-[200px]', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
let __VLS_13;
/** @ts-ignore @type { | typeof __VLS_components.SelectValue} */
SelectValue;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
    placeholder: (props.placeholder),
    selectedOption: (__VLS_ctx.selectedOption?.label),
}));
const __VLS_15 = __VLS_14({
    placeholder: (props.placeholder),
    selectedOption: (__VLS_ctx.selectedOption?.label),
}, ...__VLS_functionalComponentArgsRest(__VLS_14));
// @ts-ignore
[value, multiple, disabled, dir, cn, selectedOption,];
var __VLS_10;
let __VLS_18;
/** @ts-ignore @type { | typeof __VLS_components.SelectContent | typeof __VLS_components.SelectContent} */
SelectContent;
// @ts-ignore
const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
    position: __VLS_ctx.position,
    ...{ class: "z-2000" },
}));
const __VLS_20 = __VLS_19({
    position: __VLS_ctx.position,
    ...{ class: "z-2000" },
}, ...__VLS_functionalComponentArgsRest(__VLS_19));
/** @type {__VLS_StyleScopedClasses['z-2000']} */ ;
const { default: __VLS_23 } = __VLS_21.slots;
for (const [option] of __VLS_vFor((props.options))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
        key: (option.label),
    });
    if (option.hasOwnProperty('options')) {
        let __VLS_24;
        /** @ts-ignore @type { | typeof __VLS_components.SelectGroup | typeof __VLS_components.SelectGroup} */
        SelectGroup;
        // @ts-ignore
        const __VLS_25 = __VLS_asFunctionalComponent1(__VLS_24, new __VLS_24({}));
        const __VLS_26 = __VLS_25({}, ...__VLS_functionalComponentArgsRest(__VLS_25));
        const { default: __VLS_29 } = __VLS_27.slots;
        let __VLS_30;
        /** @ts-ignore @type { | typeof __VLS_components.SelectLabel | typeof __VLS_components.SelectLabel} */
        SelectLabel;
        // @ts-ignore
        const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({}));
        const __VLS_32 = __VLS_31({}, ...__VLS_functionalComponentArgsRest(__VLS_31));
        const { default: __VLS_35 } = __VLS_33.slots;
        (option.label);
        // @ts-ignore
        [position,];
        var __VLS_33;
        for (const [item, index] of __VLS_vFor((option.options))) {
            let __VLS_36;
            /** @ts-ignore @type { | typeof __VLS_components.SelectItem | typeof __VLS_components.SelectItem} */
            SelectItem;
            // @ts-ignore
            const __VLS_37 = __VLS_asFunctionalComponent1(__VLS_36, new __VLS_36({
                key: (index),
                value: (item.value),
                disabled: (item.disabled),
            }));
            const __VLS_38 = __VLS_37({
                key: (index),
                value: (item.value),
                disabled: (item.disabled),
            }, ...__VLS_functionalComponentArgsRest(__VLS_37));
            const { default: __VLS_41 } = __VLS_39.slots;
            (item.label);
            // @ts-ignore
            [];
            var __VLS_39;
            // @ts-ignore
            [];
        }
        // @ts-ignore
        [];
        var __VLS_27;
    }
    else {
        let __VLS_42;
        /** @ts-ignore @type { | typeof __VLS_components.SelectItem | typeof __VLS_components.SelectItem} */
        SelectItem;
        // @ts-ignore
        const __VLS_43 = __VLS_asFunctionalComponent1(__VLS_42, new __VLS_42({
            value: (option.value),
            disabled: (option.disabled),
        }));
        const __VLS_44 = __VLS_43({
            value: (option.value),
            disabled: (option.disabled),
        }, ...__VLS_functionalComponentArgsRest(__VLS_43));
        const { default: __VLS_47 } = __VLS_45.slots;
        (option.label);
        // @ts-ignore
        [];
        var __VLS_45;
    }
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_21;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
export default {};
