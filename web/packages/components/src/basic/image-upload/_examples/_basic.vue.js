import { ref } from 'vue';
import { useToast } from '../../toast';
import FaImageUpload from '../index.vue';
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
const __VLS_0 = FaImageUpload;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => response.data.url),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => response.data.url),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.onSuccess} */
    onOnSuccess: (__VLS_ctx.handleSuccess),
};
var __VLS_7;
var __VLS_3;
var __VLS_4;
// @ts-ignore
[files, handleSuccess,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
