import { onMounted, ref, useSlots, watchEffect } from 'vue';
import { ButtonGroup, ButtonGroupSeparator } from './button-group';
defineOptions({
    name: 'BuiltInButtonGroup',
});
const props = defineProps();
const slots = useSlots();
const buttonItems = ref([]);
onMounted(() => {
    // 主动捕获默认槽位中提供的所有内容
    watchEffect(() => {
        buttonItems.value = slots.default ? slots.default() : [];
    });
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
/** @ts-ignore @type { | typeof __VLS_components.ButtonGroup | typeof __VLS_components.ButtonGroup} */
ButtonGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    orientation: __VLS_ctx.orientation,
    ...{ class: (props.class) },
}));
const __VLS_2 = __VLS_1({
    orientation: __VLS_ctx.orientation,
    ...{ class: (props.class) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
for (const [item, index] of __VLS_vFor((__VLS_ctx.buttonItems))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.template)({
        key: (index),
    });
    const __VLS_7 = (item);
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({}));
    const __VLS_9 = __VLS_8({}, ...__VLS_functionalComponentArgsRest(__VLS_8));
    if (props.separator && index !== __VLS_ctx.buttonItems.length - 1) {
        let __VLS_12;
        /** @ts-ignore @type { | typeof __VLS_components.ButtonGroupSeparator} */
        ButtonGroupSeparator;
        // @ts-ignore
        const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
            orientation: (props.orientation === 'vertical' ? 'horizontal' : 'vertical'),
        }));
        const __VLS_14 = __VLS_13({
            orientation: (props.orientation === 'vertical' ? 'horizontal' : 'vertical'),
        }, ...__VLS_functionalComponentArgsRest(__VLS_13));
    }
    // @ts-ignore
    [orientation, buttonItems, buttonItems,];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
});
export default {};
