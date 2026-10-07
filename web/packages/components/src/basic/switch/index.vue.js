import Icon from '../icon/index.vue';
import { Switch } from './switch';
defineOptions({
    name: 'BuiltInSwitch',
});
const props = defineProps();
const enabled = defineModel();
async function handleChange(value) {
    if (!props.beforeChange) {
        enabled.value = value;
        return;
    }
    try {
        const result = await Promise.resolve(props.beforeChange());
        if (result) {
            enabled.value = value;
        }
    }
    catch { }
}
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Switch | typeof __VLS_components.Switch} */
Switch;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onUpdate:modelValue': {} },
    modelValue: (__VLS_ctx.enabled),
    disabled: __VLS_ctx.disabled,
}));
const __VLS_2 = __VLS_1({
    ...{ 'onUpdate:modelValue': {} },
    modelValue: (__VLS_ctx.enabled),
    disabled: __VLS_ctx.disabled,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.'update:modelValue'} */
    'onUpdate:modelValue': (__VLS_ctx.handleChange),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
{
    const { thumb: __VLS_9 } = __VLS_3.slots;
    if ((__VLS_ctx.enabled && __VLS_ctx.onIcon) || (!__VLS_ctx.enabled && __VLS_ctx.offIcon)) {
        const __VLS_10 = Icon;
        // @ts-ignore
        const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
            name: (__VLS_ctx.enabled ? __VLS_ctx.onIcon : __VLS_ctx.offIcon),
            ...{ class: "text-foreground size-3" },
        }));
        const __VLS_12 = __VLS_11({
            name: (__VLS_ctx.enabled ? __VLS_ctx.onIcon : __VLS_ctx.offIcon),
            ...{ class: "text-foreground size-3" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_11));
        /** @type {__VLS_StyleScopedClasses['text-foreground']} */ ;
        /** @type {__VLS_StyleScopedClasses['size-3']} */ ;
    }
    // @ts-ignore
    [enabled, enabled, enabled, enabled, disabled, handleChange, onIcon, onIcon, offIcon, offIcon,];
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
});
export default {};
