import { useTextDirection } from '@vueuse/core';
import { computed, useId, watch } from 'vue';
import { cn } from '#utils';
import { Label } from '../label/label';
import { RadioGroup, RadioGroupItem } from './radio-group';
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInRadioGroup',
    });
    const props = defineProps();
    const emit = defineEmits();
    const slots = defineSlots();
    const value = defineModel();
    const documentDir = useTextDirection({
        observe: true,
    });
    const dir = computed(() => props.dir ?? (documentDir.value === 'rtl' ? 'rtl' : 'ltr'));
    const baseId = useId();
    watch(value, (newValue) => {
        emit('change', newValue);
    });
    function getOptionId(option, index) {
        return option.id || `${baseId}-${index}`;
    }
    function getOptionKey(option, index) {
        if (option.id) {
            return option.id;
        }
        return typeof option.value === 'string' || typeof option.value === 'number'
            ? option.value
            : index;
    }
    function isOptionDisabled(option) {
        return Boolean(props.disabled || option.disabled);
    }
    function isOptionChecked(option) {
        return Object.is(value.value, option.value);
    }
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
    /** @ts-ignore @type { | typeof __VLS_components.RadioGroup | typeof __VLS_components.RadioGroup} */
    RadioGroup;
    // @ts-ignore
    const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
        modelValue: (__VLS_ctx.value),
        disabled: (__VLS_ctx.disabled),
        dir: (__VLS_ctx.dir),
        ...{ class: (props.class) },
    }));
    const __VLS_2 = __VLS_1({
        modelValue: (__VLS_ctx.value),
        disabled: (__VLS_ctx.disabled),
        dir: (__VLS_ctx.dir),
        ...{ class: (props.class) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_1));
    var __VLS_5;
    const { default: __VLS_6 } = __VLS_3.slots;
    for (const [option, index] of __VLS_vFor((props.options))) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            key: (__VLS_ctx.getOptionKey(option, index)),
            ...{ class: (__VLS_ctx.cn('flex gap-2', option.description ? 'items-start' : 'items-center', props.optionClass)) },
        });
        let __VLS_7;
        /** @ts-ignore @type { | typeof __VLS_components.RadioGroupItem} */
        RadioGroupItem;
        // @ts-ignore
        const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
            id: (__VLS_ctx.getOptionId(option, index)),
            value: (option.value),
            disabled: (__VLS_ctx.isOptionDisabled(option)),
            ...{ class: (__VLS_ctx.cn(props.itemClass, slots.option && 'hidden')) },
        }));
        const __VLS_9 = __VLS_8({
            id: (__VLS_ctx.getOptionId(option, index)),
            value: (option.value),
            disabled: (__VLS_ctx.isOptionDisabled(option)),
            ...{ class: (__VLS_ctx.cn(props.itemClass, slots.option && 'hidden')) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_8));
        let __VLS_12;
        /** @ts-ignore @type { | typeof __VLS_components.Label | typeof __VLS_components.Label} */
        Label;
        // @ts-ignore
        const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
            for: (__VLS_ctx.getOptionId(option, index)),
            ...{ class: (__VLS_ctx.cn('min-w-0 flex-1 cursor-pointer gap-0', option.description ? 'items-start' : 'items-center', __VLS_ctx.isOptionDisabled(option) && 'cursor-not-allowed opacity-60')) },
        }));
        const __VLS_14 = __VLS_13({
            for: (__VLS_ctx.getOptionId(option, index)),
            ...{ class: (__VLS_ctx.cn('min-w-0 flex-1 cursor-pointer gap-0', option.description ? 'items-start' : 'items-center', __VLS_ctx.isOptionDisabled(option) && 'cursor-not-allowed opacity-60')) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_13));
        const { default: __VLS_17 } = __VLS_15.slots;
        __VLS_asFunctionalSlot(slots.option)({
            id: (__VLS_ctx.getOptionId(option, index)),
            option: (option),
            checked: (__VLS_ctx.isOptionChecked(option)),
            disabled: (__VLS_ctx.isOptionDisabled(option)),
        });
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "gap-1 grid min-w-0" },
        });
        /** @type {__VLS_StyleScopedClasses['gap-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['grid']} */ ;
        /** @type {__VLS_StyleScopedClasses['min-w-0']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
            ...{ class: "truncate" },
        });
        /** @type {__VLS_StyleScopedClasses['truncate']} */ ;
        (option.label);
        if (option.description) {
            __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
                ...{ class: "text-xs text-muted-foreground leading-5 font-normal" },
            });
            /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
            /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
            /** @type {__VLS_StyleScopedClasses['leading-5']} */ ;
            /** @type {__VLS_StyleScopedClasses['font-normal']} */ ;
            (option.description);
        }
        // @ts-ignore
        [value, disabled, dir, getOptionKey, cn, cn, cn, getOptionId, getOptionId, getOptionId, isOptionDisabled, isOptionDisabled, isOptionDisabled, isOptionChecked,];
        var __VLS_15;
        // @ts-ignore
        [];
    }
    // @ts-ignore
    [];
    var __VLS_3;
    // @ts-ignore
    [];
    return {};
})()) => ({}));
export default {};
