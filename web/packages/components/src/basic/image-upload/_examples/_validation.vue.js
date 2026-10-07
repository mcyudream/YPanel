import { ref } from 'vue';
import FaIcon from '../../icon/index.vue';
import { useToast } from '../../toast';
import FaImageUpload from '../index.vue';
const toast = useToast();
const files = ref([]);
function beforeUpload(file) {
    if (!file.type.startsWith('image/')) {
        toast.error('请选择图片文件');
        return false;
    }
    if (file.size > 200 * 1024) {
        toast.error('图片大小不能超过 200KB');
        return false;
    }
    return true;
}
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
const __VLS_0 = FaImageUpload || FaImageUpload;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => `${response.data.url}?fake=${Math.random()}`),
    beforeUpload: (__VLS_ctx.beforeUpload),
    width: (200),
    height: (130),
    max: (0),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onOnSuccess': {} },
    modelValue: (__VLS_ctx.files),
    action: "/fake/upload",
    afterUpload: (response => `${response.data.url}?fake=${Math.random()}`),
    beforeUpload: (__VLS_ctx.beforeUpload),
    width: (200),
    height: (130),
    max: (0),
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
    name: "i-noto:identification-card",
    ...{ class: "opacity-50 size-50" },
}));
const __VLS_11 = __VLS_10({
    name: "i-noto:identification-card",
    ...{ class: "opacity-50 size-50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_10));
/** @type {__VLS_StyleScopedClasses['opacity-50']} */ ;
/** @type {__VLS_StyleScopedClasses['size-50']} */ ;
// @ts-ignore
[files, beforeUpload, handleSuccess,];
var __VLS_3;
var __VLS_4;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
