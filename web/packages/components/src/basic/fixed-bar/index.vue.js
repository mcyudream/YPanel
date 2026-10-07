import { onActivated, onDeactivated, onMounted, ref } from 'vue';
import { cn } from '#utils';
defineOptions({
    name: 'BuiltInFixedBar',
});
const props = defineProps();
const isActive = ref(false);
onMounted(() => {
    isActive.value = true;
});
onActivated(() => {
    isActive.value = true;
});
onDeactivated(() => {
    isActive.value = false;
});
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
/** @ts-ignore @type { | typeof __VLS_components.Teleport | typeof __VLS_components.Teleport} */
Teleport;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    to: (`#${props.position === 'top' ? 'fixed-content-before-area' : 'fixed-content-after-area'}`),
    defer: true,
    disabled: (!__VLS_ctx.isActive),
}));
const __VLS_2 = __VLS_1({
    to: (`#${props.position === 'top' ? 'fixed-content-before-area' : 'fixed-content-after-area'}`),
    defer: true,
    disabled: (!__VLS_ctx.isActive),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: (__VLS_ctx.cn('mx-auto bg-background pointer-events-auto p-4', props.class)) },
});
var __VLS_6 = {};
// @ts-ignore
[isActive, cn,];
var __VLS_3;
// @ts-ignore
var __VLS_7 = __VLS_6;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
