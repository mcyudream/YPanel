import Icon from '../icon/index.vue';
import { Alert, AlertDescription, AlertTitle } from './alert';
defineOptions({
    name: 'BuiltInAlert',
});
const props = defineProps();
const __VLS_slots = defineSlots();
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
/** @ts-ignore @type { | typeof __VLS_components.Alert | typeof __VLS_components.Alert} */
Alert;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    variant: (__VLS_ctx.variant),
    ...{ class: ([
            __VLS_ctx.icon && 'grid-cols-[calc(var(--spacing)*4)_1fr] gap-x-3 [&>[data-slot=alert-icon]]:(size-4 translate-y-0.5 text-current)',
            props.class,
        ]) },
}));
const __VLS_2 = __VLS_1({
    variant: (__VLS_ctx.variant),
    ...{ class: ([
            __VLS_ctx.icon && 'grid-cols-[calc(var(--spacing)*4)_1fr] gap-x-3 [&>[data-slot=alert-icon]]:(size-4 translate-y-0.5 text-current)',
            props.class,
        ]) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
if (__VLS_ctx.icon) {
    const __VLS_7 = Icon;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
        dataSlot: "alert-icon",
        name: (__VLS_ctx.icon),
    }));
    const __VLS_9 = __VLS_8({
        dataSlot: "alert-icon",
        name: (__VLS_ctx.icon),
    }, ...__VLS_functionalComponentArgsRest(__VLS_8));
}
if (__VLS_ctx.title) {
    let __VLS_12;
    /** @ts-ignore @type { | typeof __VLS_components.AlertTitle | typeof __VLS_components.AlertTitle} */
    AlertTitle;
    // @ts-ignore
    const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({}));
    const __VLS_14 = __VLS_13({}, ...__VLS_functionalComponentArgsRest(__VLS_13));
    const { default: __VLS_17 } = __VLS_15.slots;
    (__VLS_ctx.title);
    // @ts-ignore
    [variant, icon, icon, icon, title, title,];
    var __VLS_15;
}
if (__VLS_ctx.$slots.description || __VLS_ctx.description) {
    let __VLS_18;
    /** @ts-ignore @type { | typeof __VLS_components.AlertDescription | typeof __VLS_components.AlertDescription} */
    AlertDescription;
    // @ts-ignore
    const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({}));
    const __VLS_20 = __VLS_19({}, ...__VLS_functionalComponentArgsRest(__VLS_19));
    const { default: __VLS_23 } = __VLS_21.slots;
    __VLS_asFunctionalSlot(__VLS_slots.description)({});
    (__VLS_ctx.description);
    // @ts-ignore
    [$slots, description, description,];
    var __VLS_21;
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
