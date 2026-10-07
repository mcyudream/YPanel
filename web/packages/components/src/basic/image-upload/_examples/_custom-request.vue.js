import { ref } from 'vue';
import { useToast } from '../../toast';
import FaImageUpload from '../index.vue';
const toast = useToast();
const files = ref([]);
async function httpRequest({ file, onProgress }) {
    onProgress(30);
    await new Promise(resolve => setTimeout(resolve, 300));
    onProgress(100);
    return {
        url: URL.createObjectURL(file),
        name: file.name,
    };
}
function handleSuccess() {
    toast.success('自定义上传完成');
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
    httpRequest: (__VLS_ctx.httpRequest),
    afterUpload: (response => response.url),
    max: (3),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    httpRequest: (__VLS_ctx.httpRequest),
    afterUpload: (response => response.url),
    max: (3),
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
[files, httpRequest, handleSuccess,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
