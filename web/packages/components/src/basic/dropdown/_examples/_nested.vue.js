import FaButton from '../../button/index.vue';
import FaIcon from '../../icon/index.vue';
import { useToast } from '../../toast';
import FaDropdown from '../index.vue';
const toast = useToast();
function handleClick(text) {
    toast(text);
}
const items = [
    [
        { label: '打开', handle: () => handleClick('打开') },
        {
            label: '更多操作',
            items: [
                [
                    { label: '保存页面', handle: () => handleClick('保存页面') },
                    { label: '导出为 PDF', handle: () => handleClick('导出为 PDF') },
                ],
                [
                    { label: '复制路径', handle: () => handleClick('复制路径') },
                ],
            ],
        },
    ],
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaDropdown || FaDropdown;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    items: (__VLS_ctx.items),
}));
const __VLS_2 = __VLS_1({
    items: (__VLS_ctx.items),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
const __VLS_7 = FaButton || FaButton;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    variant: "outline",
}));
const __VLS_9 = __VLS_8({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
const { default: __VLS_12 } = __VLS_10.slots;
const __VLS_13 = FaIcon;
// @ts-ignore
const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
    name: "i-ep:caret-bottom",
}));
const __VLS_15 = __VLS_14({
    name: "i-ep:caret-bottom",
}, ...__VLS_functionalComponentArgsRest(__VLS_14));
// @ts-ignore
[items,];
var __VLS_10;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
