import { h } from 'vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../../button/index.vue';
import { useToast } from '../../toast';
import { useModal } from '../index';
const toast = useToast();
const { open } = useModal().create({
    title: '函数式调用',
    description: '通过 useModal().create() 创建弹窗。',
    content: h('div', { class: 'text-sm text-muted-foreground leading-6' }, '这里是函数式调用渲染的内容。'),
    showCancelButton: true,
    onConfirm: () => toast('确认操作'),
    onCancel: () => toast('取消操作'),
});
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
    onClick: (__VLS_ctx.open),
};
var __VLS_7;
const { default: __VLS_8 } = __VLS_3.slots;
// @ts-ignore
[open,];
var __VLS_3;
var __VLS_4;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
