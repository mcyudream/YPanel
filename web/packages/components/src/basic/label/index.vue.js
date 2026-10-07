import { Label } from './label';
defineOptions({
    name: 'BuiltInLabel',
});
const props = defineProps();
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
/** @ts-ignore @type { | typeof __VLS_components.Label | typeof __VLS_components.Label} */
Label;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ class: (props.class) },
}));
const __VLS_2 = __VLS_1({
    ...{ class: (props.class) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
if (props.label) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-muted-foreground text-nowrap" },
        ...{ style: ({ width: props.labelWidth ? (typeof props.labelWidth === 'number' ? `${props.labelWidth}px` : props.labelWidth) : undefined }) },
    });
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-nowrap']} */ ;
    (props.label);
}
var __VLS_7 = {};
var __VLS_3;
// @ts-ignore
var __VLS_8 = __VLS_7;
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
