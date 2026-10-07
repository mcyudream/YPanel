import { omit } from 'es-toolkit';
import { computed, ref, useTemplateRef } from 'vue';
import { cn } from '#utils';
import Button from '../button/index.vue';
import Icon from '../icon/index.vue';
import { Input } from './input';
import { InputGroup, InputGroupAddon } from './input-group';
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInInput',
    });
    const props = withDefaults(defineProps(), {
        type: 'text',
        align: 'inline',
    });
    const emits = defineEmits();
    const slots = defineSlots();
    const value = defineModel();
    const type = ref(props.type);
    const inputRef = useTemplateRef({});
    const isFocused = ref(false);
    const isHovered = ref(false);
    function handleClear() {
        value.value = undefined;
        emits('clear');
    }
    const __VLS_exposed = {
        ref: computed(() => inputRef.value?.ref),
    };
    defineExpose(__VLS_exposed);
    let __VLS_modelEmit;
    const __VLS_defaults = {
        type: 'text',
        align: 'inline',
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
    /** @ts-ignore @type { | typeof __VLS_components.InputGroup | typeof __VLS_components.InputGroup} */
    InputGroup;
    // @ts-ignore
    const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
        ...{ 'onMouseenter': {} },
        ...{ 'onMouseleave': {} },
        ...{ class: (__VLS_ctx.cn('w-[200px]', props.class)) },
    }));
    const __VLS_2 = __VLS_1({
        ...{ 'onMouseenter': {} },
        ...{ 'onMouseleave': {} },
        ...{ class: (__VLS_ctx.cn('w-[200px]', props.class)) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_1));
    let __VLS_5;
    const __VLS_6 = {
        /** @type {typeof __VLS_5.mouseenter} */
        onMouseenter: (...[$event]) => {
            return (__VLS_ctx.isHovered = true);
            // @ts-ignore
            [cn, isHovered,];
        },
    };
    const __VLS_7 = {
        /** @type {typeof __VLS_5.mouseleave} */
        onMouseleave: (...[$event]) => {
            return (__VLS_ctx.isHovered = false);
            // @ts-ignore
            [isHovered,];
        },
    };
    var __VLS_8;
    const { default: __VLS_9 } = __VLS_3.slots;
    let __VLS_10;
    /** @ts-ignore @type { | typeof __VLS_components.Input} */
    Input;
    // @ts-ignore
    const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
        ...{ 'onFocus': {} },
        ...{ 'onBlur': {} },
        ...({
            ...__VLS_ctx.omit(props, ['type', 'align', 'clearable', 'class', 'inputClass', 'startClass', 'endClass']),
            type: __VLS_ctx.type,
            class: __VLS_ctx.cn('w-full flex-1 border-0 rounded-none bg-transparent shadow-none ring-offset-transparent dark:bg-transparent focus-visible:ring-0 focus-visible:ring-transparent', props.inputClass),
            ...__VLS_ctx.$attrs,
        }),
        ref: "inputRef",
        modelValue: (__VLS_ctx.value),
        dataSlot: "input-group-control",
        autocomplete: "off",
    }));
    const __VLS_12 = __VLS_11({
        ...{ 'onFocus': {} },
        ...{ 'onBlur': {} },
        ...({
            ...__VLS_ctx.omit(props, ['type', 'align', 'clearable', 'class', 'inputClass', 'startClass', 'endClass']),
            type: __VLS_ctx.type,
            class: __VLS_ctx.cn('w-full flex-1 border-0 rounded-none bg-transparent shadow-none ring-offset-transparent dark:bg-transparent focus-visible:ring-0 focus-visible:ring-transparent', props.inputClass),
            ...__VLS_ctx.$attrs,
        }),
        ref: "inputRef",
        modelValue: (__VLS_ctx.value),
        dataSlot: "input-group-control",
        autocomplete: "off",
    }, ...__VLS_functionalComponentArgsRest(__VLS_11));
    let __VLS_15;
    const __VLS_16 = {
        /** @type {typeof __VLS_15.focus} */
        onFocus: (...[$event]) => {
            return (__VLS_ctx.isFocused = true);
            // @ts-ignore
            [cn, omit, type, $attrs, value, isFocused,];
        },
    };
    const __VLS_17 = {
        /** @type {typeof __VLS_15.blur} */
        onBlur: (...[$event]) => {
            return (__VLS_ctx.isFocused = false);
            // @ts-ignore
            [isFocused,];
        },
    };
    var __VLS_18;
    var __VLS_13;
    var __VLS_14;
    if (slots.start) {
        let __VLS_20;
        /** @ts-ignore @type { | typeof __VLS_components.InputGroupAddon | typeof __VLS_components.InputGroupAddon} */
        InputGroupAddon;
        // @ts-ignore
        const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({
            align: (props.align === 'inline' ? 'inline-start' : 'block-start'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.startClass)) },
        }));
        const __VLS_22 = __VLS_21({
            align: (props.align === 'inline' ? 'inline-start' : 'block-start'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.startClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_21));
        const { default: __VLS_25 } = __VLS_23.slots;
        __VLS_asFunctionalSlot(slots.start)({});
        // @ts-ignore
        [cn,];
        var __VLS_23;
    }
    if (slots.end || props.clearable || props.type === 'password') {
        let __VLS_27;
        /** @ts-ignore @type { | typeof __VLS_components.InputGroupAddon | typeof __VLS_components.InputGroupAddon} */
        InputGroupAddon;
        // @ts-ignore
        const __VLS_28 = __VLS_asFunctionalComponent1(__VLS_27, new __VLS_27({
            align: (props.align === 'inline' ? 'inline-end' : 'block-end'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.endClass)) },
        }));
        const __VLS_29 = __VLS_28({
            align: (props.align === 'inline' ? 'inline-end' : 'block-end'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.endClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_28));
        const { default: __VLS_32 } = __VLS_30.slots;
        if (!props.disabled && props.clearable && !!__VLS_ctx.value && (__VLS_ctx.isFocused || __VLS_ctx.isHovered)) {
            const __VLS_33 = Button || Button;
            // @ts-ignore
            const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({
                ...{ 'onClick': {} },
                variant: "ghost",
                size: "icon",
                type: "button",
                ...{ class: "size-6" },
            }));
            const __VLS_35 = __VLS_34({
                ...{ 'onClick': {} },
                variant: "ghost",
                size: "icon",
                type: "button",
                ...{ class: "size-6" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_34));
            let __VLS_38;
            const __VLS_39 = {
                /** @type {typeof __VLS_38.click} */
                onClick: (__VLS_ctx.handleClear),
            };
            /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
            const { default: __VLS_40 } = __VLS_36.slots;
            const __VLS_41 = Icon;
            // @ts-ignore
            const __VLS_42 = __VLS_asFunctionalComponent1(__VLS_41, new __VLS_41({
                name: "i-lucide:x",
            }));
            const __VLS_43 = __VLS_42({
                name: "i-lucide:x",
            }, ...__VLS_functionalComponentArgsRest(__VLS_42));
            // @ts-ignore
            [cn, isHovered, value, isFocused, handleClear,];
            var __VLS_36;
            var __VLS_37;
        }
        if (props.type === 'password' && !!__VLS_ctx.value) {
            const __VLS_46 = Button || Button;
            // @ts-ignore
            const __VLS_47 = __VLS_asFunctionalComponent1(__VLS_46, new __VLS_46({
                ...{ 'onClick': {} },
                variant: "ghost",
                size: "icon",
                type: "button",
                ...{ class: "size-6" },
            }));
            const __VLS_48 = __VLS_47({
                ...{ 'onClick': {} },
                variant: "ghost",
                size: "icon",
                type: "button",
                ...{ class: "size-6" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_47));
            let __VLS_51;
            const __VLS_52 = {
                /** @type {typeof __VLS_51.click} */
                onClick: (...[$event]) => {
                    if (!(slots.end || props.clearable || props.type === 'password'))
                        throw 0;
                    if (!(props.type === 'password' && !!__VLS_ctx.value))
                        throw 0;
                    return (__VLS_ctx.type = __VLS_ctx.type === 'password' ? 'text' : 'password');
                    // @ts-ignore
                    [type, type, value,];
                },
            };
            /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
            const { default: __VLS_53 } = __VLS_49.slots;
            const __VLS_54 = Icon;
            // @ts-ignore
            const __VLS_55 = __VLS_asFunctionalComponent1(__VLS_54, new __VLS_54({
                name: (__VLS_ctx.type === 'password' ? 'i-lucide:eye-off' : 'i-lucide:eye'),
            }));
            const __VLS_56 = __VLS_55({
                name: (__VLS_ctx.type === 'password' ? 'i-lucide:eye-off' : 'i-lucide:eye'),
            }, ...__VLS_functionalComponentArgsRest(__VLS_55));
            // @ts-ignore
            [type,];
            var __VLS_49;
            var __VLS_50;
        }
        __VLS_asFunctionalSlot(slots.end)({});
        // @ts-ignore
        [];
        var __VLS_30;
    }
    // @ts-ignore
    [];
    var __VLS_3;
    var __VLS_4;
    // @ts-ignore
    var __VLS_19 = __VLS_18;
    // @ts-ignore
    [];
    return {};
})()) => ({}));
export default {};
