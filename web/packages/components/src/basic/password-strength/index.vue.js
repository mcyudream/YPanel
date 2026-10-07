import { computed } from 'vue';
import Icon from '../icon/index.vue';
import Tooltip from '../tooltip/index.vue';
defineOptions({
    name: 'BuiltInPasswordStrength',
});
const props = withDefaults(defineProps(), {
    password: '',
    rules: () => [
        { label: '长度至少为8个字符', rule: value => value.length >= 8 },
        { label: '包含大写字母', rule: value => /[A-Z]/.test(value) },
        { label: '包含小写字母', rule: value => /[a-z]/.test(value) },
        { label: '包含数字', rule: value => /\d/.test(value) },
        { label: '包含特殊字符', rule: value => /[^A-Z0-9]/i.test(value) },
    ],
    colorThresholds: () => [
        { min: 0, color: 'bg-red-500' },
        { min: 1, color: 'bg-orange-500' },
        { min: 3, color: 'bg-yellow-500' },
        { min: 5, color: 'bg-green-500' },
    ],
});
const ruleStates = computed(() => props.rules.map(item => ({
    label: item.label,
    matched: item.rule(props.password),
})));
const maxStrength = computed(() => props.rules.length);
const strength = computed(() => ruleStates.value.filter(item => item.matched).length);
const progressWidth = computed(() => maxStrength.value > 0 ? `${(strength.value / maxStrength.value) * 100}%` : '0%');
const strengthColor = computed(() => {
    return [...props.colorThresholds]
        .sort((a, b) => b.min - a.min)
        .find(item => strength.value >= item.min)
        ?.color;
});
const __VLS_defaults = {
    password: '',
    rules: () => [
        { label: '长度至少为8个字符', rule: value => value.length >= 8 },
        { label: '包含大写字母', rule: value => /[A-Z]/.test(value) },
        { label: '包含小写字母', rule: value => /[a-z]/.test(value) },
        { label: '包含数字', rule: value => /\d/.test(value) },
        { label: '包含特殊字符', rule: value => /[^A-Z0-9]/i.test(value) },
    ],
    colorThresholds: () => [
        { min: 0, color: 'bg-red-500' },
        { min: 1, color: 'bg-orange-500' },
        { min: 3, color: 'bg-yellow-500' },
        { min: 5, color: 'bg-green-500' },
    ],
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex-center gap-2 w-full" },
});
/** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "rounded-full bg-border flex-1 h-2 overflow-hidden" },
});
/** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-border']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
/** @type {__VLS_StyleScopedClasses['h-2']} */ ;
/** @type {__VLS_StyleScopedClasses['overflow-hidden']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div)({
    ...{ class: (__VLS_ctx.strengthColor) },
    ...{ style: ({ width: __VLS_ctx.progressWidth }) },
    ...{ class: "h-full transition-all duration-300 ease-out" },
});
/** @type {__VLS_StyleScopedClasses['h-full']} */ ;
/** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
/** @type {__VLS_StyleScopedClasses['duration-300']} */ ;
/** @type {__VLS_StyleScopedClasses['ease-out']} */ ;
const __VLS_0 = Tooltip || Tooltip;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({}));
const __VLS_2 = __VLS_1({}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
const __VLS_6 = Icon;
// @ts-ignore
const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({
    name: "i-ri:question-line",
    ...{ class: "text-sm text-muted-foreground cursor-help" },
}));
const __VLS_8 = __VLS_7({
    name: "i-ri:question-line",
    ...{ class: "text-sm text-muted-foreground cursor-help" },
}, ...__VLS_functionalComponentArgsRest(__VLS_7));
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
/** @type {__VLS_StyleScopedClasses['cursor-help']} */ ;
{
    const { content: __VLS_11 } = __VLS_3.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "py-1" },
    });
    /** @type {__VLS_StyleScopedClasses['py-1']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.ul, __VLS_intrinsics.ul)({
        ...{ class: "text-sm text-muted-foreground space-y-1" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['space-y-1']} */ ;
    for (const [item] of __VLS_vFor((__VLS_ctx.ruleStates))) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.li, __VLS_intrinsics.li)({
            key: (item.label),
            ...{ class: "flex-center-start gap-1" },
            ...{ class: ({ 'text-green-600': item.matched }) },
        });
        /** @type {__VLS_StyleScopedClasses['flex-center-start']} */ ;
        /** @type {__VLS_StyleScopedClasses['gap-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-green-600']} */ ;
        const __VLS_12 = Icon;
        // @ts-ignore
        const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
            name: (item.matched ? 'i-carbon:checkmark' : 'i-carbon:close'),
        }));
        const __VLS_14 = __VLS_13({
            name: (item.matched ? 'i-carbon:checkmark' : 'i-carbon:close'),
        }, ...__VLS_functionalComponentArgsRest(__VLS_13));
        (item.label);
        // @ts-ignore
        [strengthColor, progressWidth, ruleStates,];
    }
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
export default {};
