import FaButton from '../../button/index.vue';
import { useToast } from '../index';
const toast = useToast();
function showToast() {
    toast('文件已移入回收站', {
        description: '你可以在 30 天内恢复该文件',
        action: {
            label: '撤销',
            onClick: () => {
                toast.success('已撤销删除');
            },
        },
    });
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaButton || FaButton;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onClick': {} },
}));
const __VLS_2 = __VLS_1({
    ...{ 'onClick': {} },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.click} */
    onClick: (__VLS_ctx.showToast),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
// @ts-ignore
[showToast,];
var __VLS_3;
var __VLS_4;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
