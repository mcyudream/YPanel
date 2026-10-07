import { computed, ref } from 'vue';
import FaRadioGroup from '../index.vue';
const value = ref('growth');
const options = [
    {
        label: '创业版',
        value: 'starter',
        description: '适合 1-10 人小团队，保留核心能力。',
    },
    {
        label: '成长版',
        value: 'growth',
        description: '适合多角色协作，支持审批与审计流程。',
    },
    {
        label: '企业版',
        value: 'enterprise',
        description: '高级安全策略与 SSO 即将开放。',
        disabled: true,
    },
];
const currentLabel = computed(() => options.find(option => option.value === value.value)?.label ?? '');
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-4 grid" },
});
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
const __VLS_0 = FaRadioGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.value),
    options: (__VLS_ctx.options),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-muted-foreground" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
(__VLS_ctx.currentLabel);
// @ts-ignore
[value, options, currentLabel,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
