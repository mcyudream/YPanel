import { ref } from 'vue';
import FaIcon from '../../icon/index.vue';
import { useToast } from '../../toast';
import FaFileUpload from '../index.vue';
const toast = useToast();
const files = ref([]);
function handleSuccess() {
    toast.success('模拟上传成功');
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaFileUpload || FaFileUpload;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => response.data.url),
    multiple: true,
}));
const __VLS_2 = __VLS_1({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => response.data.url),
    multiple: true,
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.onSuccess} */
    onOnSuccess: (__VLS_ctx.handleSuccess),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
const __VLS_9 = FaIcon;
// @ts-ignore
const __VLS_10 = __VLS_asFunctionalComponent1(__VLS_9, new __VLS_9({
    name: "i-lucide:folder-up",
    ...{ class: "text-3xl text-primary mb-2" },
}));
const __VLS_11 = __VLS_10({
    name: "i-lucide:folder-up",
    ...{ class: "text-3xl text-primary mb-2" },
}, ...__VLS_functionalComponentArgsRest(__VLS_10));
/** @type {__VLS_StyleScopedClasses['text-3xl']} */ ;
/** @type {__VLS_StyleScopedClasses['text-primary']} */ ;
/** @type {__VLS_StyleScopedClasses['mb-2']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm font-medium" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-xs text-muted-foreground mt-1" },
});
/** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
/** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
/** @type {__VLS_StyleScopedClasses['mt-1']} */ ;
// @ts-ignore
[files, handleSuccess,];
var __VLS_3;
var __VLS_4;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
