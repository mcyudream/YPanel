import { omit } from 'es-toolkit';
import { cn } from '#utils';
import { InputGroup, InputGroupAddon } from './input-group';
import { Textarea } from './textarea';
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInTextarea',
    });
    const props = withDefaults(defineProps(), {
        align: 'inline',
    });
    const slots = defineSlots();
    const value = defineModel();
    let __VLS_modelEmit;
    const __VLS_defaults = {
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
        ...{ class: (props.class) },
    }));
    const __VLS_2 = __VLS_1({
        ...{ class: (props.class) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_1));
    var __VLS_5;
    const { default: __VLS_6 } = __VLS_3.slots;
    let __VLS_7;
    /** @ts-ignore @type { | typeof __VLS_components.Textarea} */
    Textarea;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
        modelValue: (__VLS_ctx.value),
        ...({
            ...__VLS_ctx.omit(props, ['align', 'class', 'inputClass', 'startClass', 'endClass']),
            class: __VLS_ctx.cn('flex-1 resize-none rounded-none border-0 bg-transparent py-3 shadow-none focus-visible:ring-0 focus-visible:ring-transparent ring-offset-transparent dark:bg-transparent', props.inputClass),
            ...__VLS_ctx.$attrs,
        }),
        dataSlot: "input-group-control",
    }));
    const __VLS_9 = __VLS_8({
        modelValue: (__VLS_ctx.value),
        ...({
            ...__VLS_ctx.omit(props, ['align', 'class', 'inputClass', 'startClass', 'endClass']),
            class: __VLS_ctx.cn('flex-1 resize-none rounded-none border-0 bg-transparent py-3 shadow-none focus-visible:ring-0 focus-visible:ring-transparent ring-offset-transparent dark:bg-transparent', props.inputClass),
            ...__VLS_ctx.$attrs,
        }),
        dataSlot: "input-group-control",
    }, ...__VLS_functionalComponentArgsRest(__VLS_8));
    if (slots.start) {
        let __VLS_12;
        /** @ts-ignore @type { | typeof __VLS_components.InputGroupAddon | typeof __VLS_components.InputGroupAddon} */
        InputGroupAddon;
        // @ts-ignore
        const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
            align: (props.align === 'inline' ? 'inline-start' : 'block-start'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.startClass)) },
        }));
        const __VLS_14 = __VLS_13({
            align: (props.align === 'inline' ? 'inline-start' : 'block-start'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.startClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_13));
        const { default: __VLS_17 } = __VLS_15.slots;
        __VLS_asFunctionalSlot(slots.start)({});
        // @ts-ignore
        [value, omit, cn, cn, $attrs,];
        var __VLS_15;
    }
    if (slots.end) {
        let __VLS_19;
        /** @ts-ignore @type { | typeof __VLS_components.InputGroupAddon | typeof __VLS_components.InputGroupAddon} */
        InputGroupAddon;
        // @ts-ignore
        const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
            align: (props.align === 'inline' ? 'inline-end' : 'block-end'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.endClass)) },
        }));
        const __VLS_21 = __VLS_20({
            align: (props.align === 'inline' ? 'inline-end' : 'block-end'),
            ...{ class: (__VLS_ctx.cn('empty:hidden', props.endClass)) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_20));
        const { default: __VLS_24 } = __VLS_22.slots;
        __VLS_asFunctionalSlot(slots.end)({});
        // @ts-ignore
        [cn,];
        var __VLS_22;
    }
    // @ts-ignore
    [];
    var __VLS_3;
    // @ts-ignore
    [];
    return {};
})()) => ({}));
export default {};
