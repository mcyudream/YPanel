import { Tag } from './tag';
defineOptions({
    name: 'BuiltInTag',
});
const props = defineProps();
const emit = defineEmits();
function handleClose(event) {
    emit('close', event);
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
/** @ts-ignore @type { | typeof __VLS_components.Tag | typeof __VLS_components.Tag} */
Tag;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onClose': {} },
    variant: __VLS_ctx.variant,
    icon: __VLS_ctx.icon,
    closable: __VLS_ctx.closable,
    ...{ class: (props.class) },
}));
const __VLS_2 = __VLS_1({
    ...{ 'onClose': {} },
    variant: __VLS_ctx.variant,
    icon: __VLS_ctx.icon,
    closable: __VLS_ctx.closable,
    ...{ class: (props.class) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.close} */
    onClose: (__VLS_ctx.handleClose),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
var __VLS_9 = {};
// @ts-ignore
[variant, icon, closable, handleClose,];
var __VLS_3;
var __VLS_4;
// @ts-ignore
var __VLS_10 = __VLS_9;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
const __VLS_export = {};
export default {};
