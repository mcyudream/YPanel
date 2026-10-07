import { ref } from 'vue';
import FaTabs from '../index.vue';
const activeTab = ref('overview');
const list = [
    { label: '总览', value: 'overview', class: 'rounded-md data-[state=active]:bg-primary data-[state=active]:text-primary-foreground' },
    { label: '趋势', value: 'trend', class: 'rounded-md data-[state=active]:bg-primary data-[state=active]:text-primary-foreground' },
    { label: '明细', value: 'detail', class: 'rounded-md data-[state=active]:bg-primary data-[state=active]:text-primary-foreground' },
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaTabs || FaTabs;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.activeTab),
    list: (__VLS_ctx.list),
    ...{ class: "p-3 border rounded-lg w-96" },
    listClass: "gap-2 rounded-md bg-transparent p-0",
    contentClass: "rounded-md bg-muted/50 p-4 text-sm",
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.activeTab),
    list: (__VLS_ctx.list),
    ...{ class: "p-3 border rounded-lg w-96" },
    listClass: "gap-2 rounded-md bg-transparent p-0",
    contentClass: "rounded-md bg-muted/50 p-4 text-sm",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
/** @type {__VLS_StyleScopedClasses['p-3']} */ ;
/** @type {__VLS_StyleScopedClasses['border']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
/** @type {__VLS_StyleScopedClasses['w-96']} */ ;
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { overview: __VLS_7 } = __VLS_3.slots;
    // @ts-ignore
    [activeTab, list,];
}
{
    const { trend: __VLS_8 } = __VLS_3.slots;
    // @ts-ignore
    [];
}
{
    const { detail: __VLS_9 } = __VLS_3.slots;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
