import { ref } from 'vue';
import { useModal } from '../../modal';
import FaSwitch from '../index.vue';
const checked = ref(false);
const modal = useModal();
function handleBeforeChange() {
    return new Promise((resolve) => {
        modal.confirm({
            title: '确认信息',
            content: '确认要切换当前状态吗？',
            onConfirm: () => {
                resolve(true);
            },
            onCancel: () => {
                resolve(false);
            },
        });
    });
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaSwitch;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    modelValue: (__VLS_ctx.checked),
    beforeChange: (__VLS_ctx.handleBeforeChange),
}));
const __VLS_2 = __VLS_1({
    modelValue: (__VLS_ctx.checked),
    beforeChange: (__VLS_ctx.handleBeforeChange),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
var __VLS_3;
// @ts-ignore
[checked, handleBeforeChange,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
