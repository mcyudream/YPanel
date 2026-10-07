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
        { label: '打开', icon: 'i-lucide:folder-open', handle: () => handleClick('打开') },
        { label: '重命名', icon: 'i-lucide:pencil', handle: () => handleClick('重命名') },
        { label: '复制链接', icon: 'i-lucide:link', handle: () => handleClick('复制链接') },
    ],
    [
        { label: '下载', icon: 'i-lucide:download', handle: () => handleClick('下载') },
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
