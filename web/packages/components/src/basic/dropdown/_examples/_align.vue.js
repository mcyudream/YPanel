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
        { label: '个人设置', icon: 'i-lucide:user', handle: () => handleClick('个人设置') },
        { label: '账号安全', icon: 'i-lucide:shield', handle: () => handleClick('账号安全') },
    ],
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-wrap gap-2" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const __VLS_0 = FaDropdown || FaDropdown;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    items: (__VLS_ctx.items),
    align: "start",
}));
const __VLS_2 = __VLS_1({
    items: (__VLS_ctx.items),
    align: "start",
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
const __VLS_6 = FaButton || FaButton;
// @ts-ignore
const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({
    variant: "outline",
}));
const __VLS_8 = __VLS_7({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_7));
const { default: __VLS_11 } = __VLS_9.slots;
const __VLS_12 = FaIcon;
// @ts-ignore
const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
    name: "i-ep:caret-bottom",
}));
const __VLS_14 = __VLS_13({
    name: "i-ep:caret-bottom",
}, ...__VLS_functionalComponentArgsRest(__VLS_13));
// @ts-ignore
[items,];
var __VLS_9;
// @ts-ignore
[];
var __VLS_3;
const __VLS_17 = FaDropdown || FaDropdown;
// @ts-ignore
const __VLS_18 = __VLS_asFunctionalComponent1(__VLS_17, new __VLS_17({
    items: (__VLS_ctx.items),
    align: "center",
}));
const __VLS_19 = __VLS_18({
    items: (__VLS_ctx.items),
    align: "center",
}, ...__VLS_functionalComponentArgsRest(__VLS_18));
const { default: __VLS_22 } = __VLS_20.slots;
const __VLS_23 = FaButton || FaButton;
// @ts-ignore
const __VLS_24 = __VLS_asFunctionalComponent1(__VLS_23, new __VLS_23({
    variant: "outline",
}));
const __VLS_25 = __VLS_24({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_24));
const { default: __VLS_28 } = __VLS_26.slots;
const __VLS_29 = FaIcon;
// @ts-ignore
const __VLS_30 = __VLS_asFunctionalComponent1(__VLS_29, new __VLS_29({
    name: "i-ep:caret-bottom",
}));
const __VLS_31 = __VLS_30({
    name: "i-ep:caret-bottom",
}, ...__VLS_functionalComponentArgsRest(__VLS_30));
// @ts-ignore
[items,];
var __VLS_26;
// @ts-ignore
[];
var __VLS_20;
const __VLS_34 = FaDropdown || FaDropdown;
// @ts-ignore
const __VLS_35 = __VLS_asFunctionalComponent1(__VLS_34, new __VLS_34({
    items: (__VLS_ctx.items),
    align: "end",
}));
const __VLS_36 = __VLS_35({
    items: (__VLS_ctx.items),
    align: "end",
}, ...__VLS_functionalComponentArgsRest(__VLS_35));
const { default: __VLS_39 } = __VLS_37.slots;
const __VLS_40 = FaButton || FaButton;
// @ts-ignore
const __VLS_41 = __VLS_asFunctionalComponent1(__VLS_40, new __VLS_40({
    variant: "outline",
}));
const __VLS_42 = __VLS_41({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_41));
const { default: __VLS_45 } = __VLS_43.slots;
const __VLS_46 = FaIcon;
// @ts-ignore
const __VLS_47 = __VLS_asFunctionalComponent1(__VLS_46, new __VLS_46({
    name: "i-ep:caret-bottom",
}));
const __VLS_48 = __VLS_47({
    name: "i-ep:caret-bottom",
}, ...__VLS_functionalComponentArgsRest(__VLS_47));
// @ts-ignore
[items,];
var __VLS_43;
// @ts-ignore
[];
var __VLS_37;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
