import { ref } from 'vue';
import FaFileUpload from '../index.vue';
const files = ref([]);
async function httpRequest({ file, onProgress }) {
    onProgress(50);
    await new Promise(resolve => setTimeout(resolve, 200));
    onProgress(100);
    return {
        url: URL.createObjectURL(file),
        name: file.name,
    };
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaFileUpload;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.files),
    directory: true,
    max: (0),
    httpRequest: (__VLS_ctx.httpRequest),
    afterUpload: (response => response.url),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.files),
    directory: true,
    max: (0),
    httpRequest: (__VLS_ctx.httpRequest),
    afterUpload: (response => response.url),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[files, httpRequest,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
