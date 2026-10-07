import Icon from '../icon/index.vue';
defineOptions({
    name: 'BuiltInSearchBar',
});
const __VLS_props = withDefaults(defineProps(), {
    showToggle: true,
    background: false,
});
const emits = defineEmits();
const fold = defineModel('fold', {
    default: true,
});
function toggle() {
    fold.value = !fold.value;
    emits('toggle', fold.value);
}
const __VLS_defaultModels = {
    'fold': true,
};
let __VLS_modelEmit;
const __VLS_defaults = {
    showToggle: true,
    background: false,
};
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
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "relative" },
    ...{ class: ({
            'py-4': __VLS_ctx.showToggle,
            'px-4 bg-secondary transition': __VLS_ctx.background,
        }) },
});
/** @type {__VLS_StyleScopedClasses['relative']} */ ;
/** @type {__VLS_StyleScopedClasses['py-4']} */ ;
/** @type {__VLS_StyleScopedClasses['px-4']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-secondary']} */ ;
/** @type {__VLS_StyleScopedClasses['transition']} */ ;
var __VLS_0 = {
    fold: (__VLS_ctx.fold),
    toggle: (__VLS_ctx.toggle),
};
if (__VLS_ctx.showToggle) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-center w-full translate-y-1/2 bottom-0 left-0 absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['text-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['translate-y-1/2']} */ ;
    /** @type {__VLS_StyleScopedClasses['bottom-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['left-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (__VLS_ctx.toggle) },
        ...{ class: "text-xs font-medium px-2 outline-none border-size-0 rounded bg-secondary inline-flex h-5 cursor-pointer select-none items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
    /** @type {__VLS_StyleScopedClasses['px-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['outline-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['border-size-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-secondary']} */ ;
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['h-5']} */ ;
    /** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
    /** @type {__VLS_StyleScopedClasses['select-none']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    const __VLS_2 = Icon;
    // @ts-ignore
    const __VLS_3 = __VLS_asFunctionalComponent1(__VLS_2, new __VLS_2({
        name: (__VLS_ctx.fold ? 'i-ep:caret-bottom' : 'i-ep:caret-top'),
    }));
    const __VLS_4 = __VLS_3({
        name: (__VLS_ctx.fold ? 'i-ep:caret-bottom' : 'i-ep:caret-top'),
    }, ...__VLS_functionalComponentArgsRest(__VLS_3));
}
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[showToggle, showToggle, background, fold, fold, toggle, toggle,];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
