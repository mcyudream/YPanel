import { ref } from 'vue';
import { cn } from '#utils';
import Button from '../button/index.vue';
import Icon from '../icon/index.vue';
defineOptions({
    name: 'BuiltInPageMain',
});
const props = withDefaults(defineProps(), {
    title: '',
    collaspe: false,
    height: '',
});
const slots = defineSlots();
const isCollaspe = ref(props.collaspe);
function handleCollaspe() {
    isCollaspe.value = !isCollaspe.value;
}
const __VLS_defaults = {
    title: '',
    collaspe: false,
    height: '',
};
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: (__VLS_ctx.cn('m-4 flex flex-col overflow-hidden rounded-lg border bg-card transition-[background-color,border-color]', {
            'overflow-hidden': __VLS_ctx.collaspe,
        }, props.class)) },
});
if (!!slots.title || __VLS_ctx.title) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: (__VLS_ctx.cn('px-4 py-4 my--2 bg-muted text-muted-foreground rounded-t-lg text-sm', props.titleClass)) },
    });
    __VLS_asFunctionalSlot(slots.title)({});
    (__VLS_ctx.title);
}
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: (__VLS_ctx.cn('group/pagemain relative h-[calc-size(auto,size)] bg-inherit p-4 rounded-lg transition-height', {
            'border-t': !!slots.title || __VLS_ctx.title,
            'overflow-hidden': __VLS_ctx.collaspe,
            'mask-b-from-12': __VLS_ctx.isCollaspe,
        }, props.mainClass)) },
    ...{ style: ({
            height: __VLS_ctx.isCollaspe ? __VLS_ctx.height : '',
        }) },
});
__VLS_asFunctionalSlot(slots['default'])({});
if (__VLS_ctx.collaspe) {
    const __VLS_2 = Button || Button;
    // @ts-ignore
    const __VLS_3 = __VLS_asFunctionalComponent1(__VLS_2, new __VLS_2({
        ...{ 'onClick': {} },
        variant: "link",
        size: "icon",
        ...{ class: "opacity-0 transition-all inset-b-0 inset-s-1/2 absolute group-hover/pagemain:opacity-100 -translate-x-1/2" },
        ...{ class: ({ 'rotate-x-180': !__VLS_ctx.isCollaspe }) },
    }));
    const __VLS_4 = __VLS_3({
        ...{ 'onClick': {} },
        variant: "link",
        size: "icon",
        ...{ class: "opacity-0 transition-all inset-b-0 inset-s-1/2 absolute group-hover/pagemain:opacity-100 -translate-x-1/2" },
        ...{ class: ({ 'rotate-x-180': !__VLS_ctx.isCollaspe }) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_3));
    let __VLS_7;
    const __VLS_8 = {
        /** @type {typeof __VLS_7.click} */
        onClick: (__VLS_ctx.handleCollaspe),
    };
    /** @type {__VLS_StyleScopedClasses['opacity-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-b-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-s-1/2']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    /** @type {__VLS_StyleScopedClasses['group-hover/pagemain:opacity-100']} */ ;
    /** @type {__VLS_StyleScopedClasses['-translate-x-1/2']} */ ;
    /** @type {__VLS_StyleScopedClasses['rotate-x-180']} */ ;
    const { default: __VLS_9 } = __VLS_5.slots;
    const __VLS_10 = Icon;
    // @ts-ignore
    const __VLS_11 = __VLS_asFunctionalComponent1(__VLS_10, new __VLS_10({
        name: "i-material-symbols:arrow-drop-down-rounded",
        ...{ class: "text-xl" },
    }));
    const __VLS_12 = __VLS_11({
        name: "i-material-symbols:arrow-drop-down-rounded",
        ...{ class: "text-xl" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_11));
    /** @type {__VLS_StyleScopedClasses['text-xl']} */ ;
    // @ts-ignore
    [cn, cn, cn, collaspe, collaspe, collaspe, title, title, title, isCollaspe, isCollaspe, isCollaspe, height, handleCollaspe,];
    var __VLS_5;
    var __VLS_6;
}
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
