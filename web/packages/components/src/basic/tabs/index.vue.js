import { useTextDirection } from '@vueuse/core';
import { ref, watch } from 'vue';
import { cn } from '#utils';
import Icon from '../icon/index.vue';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs';
defineOptions({
    name: 'BuiltInTabs',
});
const props = defineProps();
const emits = defineEmits();
const dir = useTextDirection({
    observe: true,
});
const activeTab = ref(props.modelValue);
watch(() => props.modelValue, (newValue) => {
    activeTab.value = newValue;
});
function handleChange(value) {
    activeTab.value = value;
    emits('update:modelValue', value);
}
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
/** @ts-ignore @type { | typeof __VLS_components.Tabs | typeof __VLS_components.Tabs} */
Tabs;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onUpdate:modelValue': {} },
    modelValue: (__VLS_ctx.activeTab),
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
    ...{ class: (__VLS_ctx.cn('flex flex-col', props.class)) },
}));
const __VLS_2 = __VLS_1({
    ...{ 'onUpdate:modelValue': {} },
    modelValue: (__VLS_ctx.activeTab),
    dir: (__VLS_ctx.dir === 'ltr' ? 'ltr' : 'rtl'),
    ...{ class: (__VLS_ctx.cn('flex flex-col', props.class)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.'update:modelValue'} */
    'onUpdate:modelValue': (__VLS_ctx.handleChange),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
let __VLS_9;
/** @ts-ignore @type { | typeof __VLS_components.TabsList | typeof __VLS_components.TabsList} */
TabsList;
// @ts-ignore
const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
    ...{ class: (__VLS_ctx.cn('flex items-center justify-center', props.listClass)) },
}));
const __VLS_11 = __VLS_10({
    ...{ class: (__VLS_ctx.cn('flex items-center justify-center', props.listClass)) },
}, ...__VLS_functionalComponentArgsRest(__VLS_10));
const { default: __VLS_14 } = __VLS_12.slots;
for (const [item] of __VLS_vFor((__VLS_ctx.list))) {
    let __VLS_15;
    /** @ts-ignore @type { | typeof __VLS_components.TabsTrigger | typeof __VLS_components.TabsTrigger} */
    TabsTrigger;
    // @ts-ignore
    const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({
        key: (item.value),
        value: (item.value),
        ...{ class: (__VLS_ctx.cn('w-full', item.class)) },
    }));
    const __VLS_17 = __VLS_16({
        key: (item.value),
        value: (item.value),
        ...{ class: (__VLS_ctx.cn('w-full', item.class)) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_16));
    const { default: __VLS_20 } = __VLS_18.slots;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex-center gap-1" },
    });
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-1']} */ ;
    if (item.icon) {
        const __VLS_21 = Icon;
        // @ts-ignore
        const __VLS_22 = __VLS_asFunctionalComponent1(__VLS_21, new __VLS_21({
            name: (item.icon),
            ...{ class: "flex-shrink-0" },
        }));
        const __VLS_23 = __VLS_22({
            name: (item.icon),
            ...{ class: "flex-shrink-0" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_22));
        /** @type {__VLS_StyleScopedClasses['flex-shrink-0']} */ ;
    }
    (item.label);
    // @ts-ignore
    [activeTab, dir, cn, cn, cn, handleChange, list,];
    var __VLS_18;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_12;
for (const [item] of __VLS_vFor((__VLS_ctx.list))) {
    let __VLS_26;
    /** @ts-ignore @type { | typeof __VLS_components.TabsContent | typeof __VLS_components.TabsContent} */
    TabsContent;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        key: (item.value),
        value: (item.value),
        ...{ class: (props.contentClass) },
    }));
    const __VLS_28 = __VLS_27({
        key: (item.value),
        value: (item.value),
        ...{ class: (props.contentClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    const { default: __VLS_31 } = __VLS_29.slots;
    var __VLS_32 = {};
    var __VLS_33 = __VLS_tryAsConstant(item.value);
    // @ts-ignore
    [list,];
    var __VLS_29;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
// @ts-ignore
var __VLS_34 = __VLS_33, __VLS_35 = __VLS_32;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
