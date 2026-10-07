import { ref } from 'vue';
import { useToast } from '../../toast';
import FaFileUpload from '../index.vue';
const toast = useToast();
const files = ref([
    { name: 'logo.svg', size: 1024 * 1024, url: 'https://fantastic-admin.hurui.me/logo.svg' },
]);
function beforeUpload(file) {
    const isPng = file.type === 'image/png';
    const isLt200K = file.size <= 200 * 1024;
    if (!isPng) {
        toast.error('只能上传 PNG 文件');
        return false;
    }
    if (!isLt200K) {
        toast.error('文件大小不能超过 200KB');
        return false;
    }
    return true;
}
function handleSuccess() {
    toast.success('模拟上传成功');
}
function handleClick(fileItem) {
    toast.info(fileItem.name, {
        description: fileItem.url,
    });
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
    ...{ 'onOnSuccess': {} },
    ...{ 'onOnClick': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => response.data.url),
    beforeUpload: (__VLS_ctx.beforeUpload),
    multiple: true,
    max: (5),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onOnSuccess': {} },
    ...{ 'onOnClick': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => response.data.url),
    beforeUpload: (__VLS_ctx.beforeUpload),
    multiple: true,
    max: (5),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.onSuccess} */
    onOnSuccess: (__VLS_ctx.handleSuccess),
};
const __VLS_7 = {
    /** @type {typeof __VLS_5.onClick} */
    onOnClick: (__VLS_ctx.handleClick),
};
var __VLS_8;
var __VLS_3;
var __VLS_4;
// @ts-ignore
[files, beforeUpload, handleSuccess, handleClick,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
