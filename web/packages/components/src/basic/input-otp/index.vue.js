import { computed, useId } from 'vue';
import { REGEXP_ONLY_CHARS, REGEXP_ONLY_DIGITS, REGEXP_ONLY_DIGITS_AND_CHARS } from 'vue-input-otp';
import { InputOTP, InputOTPGroup, InputOTPSeparator, InputOTPSlot } from './input-otp';
defineOptions({
    name: 'BuiltInInputOTP',
});
const props = withDefaults(defineProps(), {
    length: 6,
    separator: () => [],
});
const emit = defineEmits();
const modelValue = defineModel();
const id = useId();
const pattern = computed(() => {
    switch (props.pattern) {
        case 'only-chars':
            return REGEXP_ONLY_CHARS;
        case 'only-digits':
            return REGEXP_ONLY_DIGITS;
        case 'only-digits-and-chars':
            return REGEXP_ONLY_DIGITS_AND_CHARS;
        default:
            return props.pattern;
    }
});
const groups = computed(() => {
    const result = [];
    let start = 0;
    for (const value of props.separator) {
        const groupLength = Math.trunc(value);
        if (groupLength <= 0) {
            continue;
        }
        if (start >= props.length) {
            break;
        }
        result.push({
            start,
            length: Math.min(groupLength, props.length - start),
        });
        start += groupLength;
    }
    if (start < props.length) {
        result.push({
            start,
            length: props.length - start,
        });
    }
    return result;
});
let __VLS_modelEmit;
const __VLS_defaults = {
    length: 6,
    separator: () => [],
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
/** @ts-ignore @type { | typeof __VLS_components.InputOTP | typeof __VLS_components.InputOTP} */
InputOTP;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onInput': {} },
    ...{ 'onComplete': {} },
    id: __VLS_ctx.id,
    modelValue: (__VLS_ctx.modelValue),
    maxlength: (__VLS_ctx.length),
    pattern: __VLS_ctx.pattern,
    disabled: __VLS_ctx.disabled,
}));
const __VLS_2 = __VLS_1({
    ...{ 'onInput': {} },
    ...{ 'onComplete': {} },
    id: __VLS_ctx.id,
    modelValue: (__VLS_ctx.modelValue),
    maxlength: (__VLS_ctx.length),
    pattern: __VLS_ctx.pattern,
    disabled: __VLS_ctx.disabled,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.input} */
    onInput: (...[$event]) => {
        return (__VLS_ctx.emit('input', $event));
        // @ts-ignore
        [id, modelValue, length, pattern, disabled, emit,];
    },
};
const __VLS_7 = {
    /** @type {typeof __VLS_5.complete} */
    onComplete: (...[$event]) => {
        return (__VLS_ctx.emit('complete', $event));
        // @ts-ignore
        [emit,];
    },
};
var __VLS_8;
const { default: __VLS_9 } = __VLS_3.slots;
for (const [group, groupIndex] of __VLS_vFor((__VLS_ctx.groups))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
        key: (group.start),
    });
    let __VLS_10;
    /** @ts-ignore @type { | typeof __VLS_components.InputOTPGroup | typeof __VLS_components.InputOTPGroup} */
    InputOTPGroup;
    // @ts-ignore
    const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({}));
    const __VLS_12 = __VLS_11({}, ...__VLS_functionalComponentArgsRest(__VLS_11));
    const { default: __VLS_15 } = __VLS_13.slots;
    for (const [item] of __VLS_vFor((group.length))) {
        let __VLS_16;
        /** @ts-ignore @type { | typeof __VLS_components.InputOTPSlot} */
        InputOTPSlot;
        // @ts-ignore
        const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
            key: (group.start + item),
            index: (group.start + item - 1),
        }));
        const __VLS_18 = __VLS_17({
            key: (group.start + item),
            index: (group.start + item - 1),
        }, ...__VLS_functionalComponentArgsRest(__VLS_17));
        // @ts-ignore
        [groups,];
    }
    // @ts-ignore
    [];
    var __VLS_13;
    if (groupIndex < __VLS_ctx.groups.length - 1) {
        let __VLS_21;
        /** @ts-ignore @type { | typeof __VLS_components.InputOTPSeparator} */
        InputOTPSeparator;
        // @ts-ignore
        const __VLS_22 = __VLS_asFunctionalComponent1(__VLS_21, new __VLS_21({}));
        const __VLS_23 = __VLS_22({}, ...__VLS_functionalComponentArgsRest(__VLS_22));
    }
    // @ts-ignore
    [groups,];
}
// @ts-ignore
[];
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
