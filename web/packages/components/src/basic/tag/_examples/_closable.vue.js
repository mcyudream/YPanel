import { ref } from 'vue';
import FaTag from '../index.vue';
const tags = ref([
    { id: 1, label: '标签一', variant: 'default' },
    { id: 2, label: '标签二', variant: 'destructive' },
    { id: 3, label: '标签三', variant: 'outline' },
    { id: 4, label: '标签四', variant: 'secondary' },
]);
function handleClose(id) {
    tags.value = tags.value.filter(tag => tag.id !== id);
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-wrap gap-4" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
for (const [tag] of __VLS_vFor((__VLS_ctx.tags))) {
    const __VLS_0 = FaTag || FaTag;
    // @ts-ignore
    const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
        ...{ 'onClose': {} },
        key: (tag.id),
        variant: (tag.variant),
        closable: true,
    }));
    const __VLS_2 = __VLS_1({
        ...{ 'onClose': {} },
        key: (tag.id),
        variant: (tag.variant),
        closable: true,
    }, ...__VLS_functionalComponentArgsRest(__VLS_1));
    let __VLS_5;
    const __VLS_6 = {
        /** @type {typeof __VLS_5.close} */
        onClose: (...[$event]) => {
            return (__VLS_ctx.handleClose(tag.id));
            // @ts-ignore
            [tags, handleClose,];
        },
    };
    const { default: __VLS_7 } = __VLS_3.slots;
    (tag.label);
    // @ts-ignore
    [];
    var __VLS_3;
    var __VLS_4;
    // @ts-ignore
    [];
}
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
