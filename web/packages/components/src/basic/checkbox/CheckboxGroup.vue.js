import { useTextDirection } from '@vueuse/core';
import { computed, useId, watch } from 'vue';
import { cn } from '#utils';
import { Label } from '../label/label';
import { Checkbox } from './checkbox';
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInCheckboxGroup',
    });
    const props = defineProps();
    const emit = defineEmits();
    const slots = defineSlots();
    const value = defineModel({
        default: () => [],
    });
    const documentDir = useTextDirection({
        observe: true,
    });
    const dir = computed(() => props.dir ?? (documentDir.value === 'rtl' ? 'rtl' : 'ltr'));
    const baseId = useId();
    const checkedCount = computed(() => value.value.length);
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
        if (props.disabled || option.disabled) {
            return true;
        }
        const checked = isOptionChecked(option);
        if (checked && props.min !== undefined && checkedCount.value <= props.min) {
            return true;
        }
        if (!checked && props.max !== undefined && checkedCount.value >= props.max) {
            return true;
        }
        return false;
    }
    function findOptionIndex(option) {
        return value.value.findIndex(item => Object.is(item, option.value));
    }
    function isOptionChecked(option) {
        return findOptionIndex(option) > -1;
    }
    function updateOptionChecked(option, checked) {
        const nextValue = [...value.value];
        const optionIndex = nextValue.findIndex(item => Object.is(item, option.value));
        if (checked) {
            if (optionIndex === -1) {
                nextValue.push(option.value);
            }
        }
        else if (optionIndex > -1) {
            nextValue.splice(optionIndex, 1);
        }
        value.value = nextValue;
    }
    function handleOptionModelValueChange(option, checked) {
        if (isOptionDisabled(option)) {
            return;
        }
        updateOptionChecked(option, checked === true);
    }
    function handleCustomOptionClick(option) {
        if (!slots.option || isOptionDisabled(option)) {
            return;
        }
        updateOptionChecked(option, !isOptionChecked(option));
    }
    function handleCustomOptionKeydown(option) {
        if (!slots.option || isOptionDisabled(option)) {
            return;
        }
        updateOptionChecked(option, !isOptionChecked(option));
    }
    const __VLS_defaultModels = {
        'modelValue': () => [],
    };
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
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        role: "group",
        dir: (__VLS_ctx.dir),
        ...{ class: (__VLS_ctx.cn('grid gap-3', props.class)) },
    });
    for (const [option, index] of __VLS_vFor((props.options))) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ onClick: (...[$event]) => {
                    return (__VLS_ctx.handleCustomOptionClick(option));
                    // @ts-ignore
                    [dir, cn, handleCustomOptionClick,];
                } },
            ...{ onKeydown: (...[$event]) => {
                    return (__VLS_ctx.handleCustomOptionKeydown(option));
                    // @ts-ignore
                    [handleCustomOptionKeydown,];
                } },
            ...{ onKeydown: (...[$event]) => {
                    return (__VLS_ctx.handleCustomOptionKeydown(option));
                    // @ts-ignore
                    [handleCustomOptionKeydown,];
                } },
            key: (__VLS_ctx.getOptionKey(option, index)),
            tabindex: (slots.option && !__VLS_ctx.isOptionDisabled(option) ? 0 : undefined),
            role: (slots.option ? 'checkbox' : undefined),
            'aria-checked': (slots.option ? `${__VLS_ctx.isOptionChecked(option)}` : undefined),
            'aria-disabled': (slots.option ? `${__VLS_ctx.isOptionDisabled(option)}` : undefined),
            ...{ class: (__VLS_ctx.cn('flex gap-2', option.description ? 'items-start' : 'items-center', slots.option && 'outline-none focus-visible:ring-ring/50 focus-visible:ring-3 rounded-xl', props.optionClass)) },
        });
        let __VLS_0;
        /** @ts-ignore @type { | typeof __VLS_components.Checkbox} */
        Checkbox;
        // @ts-ignore
        const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
            ...{ 'onUpdate:modelValue': {} },
            id: (__VLS_ctx.getOptionId(option, index)),
            modelValue: (__VLS_ctx.isOptionChecked(option)),
            disabled: (__VLS_ctx.isOptionDisabled(option)),
            ...{ class: (__VLS_ctx.cn(props.itemClass, slots.option && 'hidden')) },
        }));
        const __VLS_2 = __VLS_1({
            ...{ 'onUpdate:modelValue': {} },
            id: (__VLS_ctx.getOptionId(option, index)),
            modelValue: (__VLS_ctx.isOptionChecked(option)),
            disabled: (__VLS_ctx.isOptionDisabled(option)),
            ...{ class: (__VLS_ctx.cn(props.itemClass, slots.option && 'hidden')) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_1));
        let __VLS_5;
        const __VLS_6 = {
            /** @type {typeof __VLS_5.'update:modelValue'} */
            'onUpdate:modelValue': (...[$event]) => {
                return (__VLS_ctx.handleOptionModelValueChange(option, $event));
                // @ts-ignore
                [cn, cn, getOptionKey, isOptionDisabled, isOptionDisabled, isOptionDisabled, isOptionChecked, isOptionChecked, getOptionId, handleOptionModelValueChange,];
            },
        };
        var __VLS_3;
        var __VLS_4;
        let __VLS_7;
        /** @ts-ignore @type { | typeof __VLS_components.Label | typeof __VLS_components.Label} */
        Label;
        // @ts-ignore
        const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
            for: (slots.option ? undefined : __VLS_ctx.getOptionId(option, index)),
            ...{ class: (__VLS_ctx.cn('min-w-0 flex-1 cursor-pointer gap-0', option.description ? 'items-start' : 'items-center', __VLS_ctx.isOptionDisabled(option) && 'cursor-not-allowed opacity-60', props.labelClass)) },
        }));
        const __VLS_9 = __VLS_8({
            for: (slots.option ? undefined : __VLS_ctx.getOptionId(option, index)),
            ...{ class: (__VLS_ctx.cn('min-w-0 flex-1 cursor-pointer gap-0', option.description ? 'items-start' : 'items-center', __VLS_ctx.isOptionDisabled(option) && 'cursor-not-allowed opacity-60', props.labelClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_8));
        const { default: __VLS_12 } = __VLS_10.slots;
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
        [cn, isOptionDisabled, isOptionDisabled, isOptionChecked, getOptionId, getOptionId,];
        var __VLS_10;
        // @ts-ignore
        [];
    }
    // @ts-ignore
    [];
    return {};
})()) => ({}));
export default {};
