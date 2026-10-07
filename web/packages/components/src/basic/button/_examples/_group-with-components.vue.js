import { shallowRef } from 'vue';
import FaDropdown from '../../dropdown/index.vue';
import FaIcon from '../../icon/index.vue';
import FaInput from '../../input/index.vue';
import FaSelect from '../../select/index.vue';
import FaButtonGroup from '../ButtonGroup.vue';
// 组件实际使用时无需手动导入，框架会自动导入
import FaButton from '../index.vue';
const currency = shallowRef('CNY');
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-col gap-4 items-start" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-col']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['items-start']} */ ;
const __VLS_0 = FaButtonGroup || FaButtonGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({}));
const __VLS_2 = __VLS_1({}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
const __VLS_6 = FaInput;
// @ts-ignore
const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({}));
const __VLS_8 = __VLS_7({}, ...__VLS_functionalComponentArgsRest(__VLS_7));
const __VLS_11 = FaButton || FaButton;
// @ts-ignore
const __VLS_12 = __VLS_asFunctionalComponent1(__VLS_11, new __VLS_11({
    variant: "outline",
    size: "icon",
}));
const __VLS_13 = __VLS_12({
    variant: "outline",
    size: "icon",
}, ...__VLS_functionalComponentArgsRest(__VLS_12));
const { default: __VLS_16 } = __VLS_14.slots;
const __VLS_17 = FaIcon;
// @ts-ignore
const __VLS_18 = __VLS_asFunctionalComponent1(__VLS_17, new __VLS_17({
    name: "i-ep:search",
}));
const __VLS_19 = __VLS_18({
    name: "i-ep:search",
}, ...__VLS_functionalComponentArgsRest(__VLS_18));
var __VLS_14;
var __VLS_3;
const __VLS_22 = FaButtonGroup || FaButtonGroup;
// @ts-ignore
const __VLS_23 = __VLS_asFunctionalComponent1(__VLS_22, new __VLS_22({}));
const __VLS_24 = __VLS_23({}, ...__VLS_functionalComponentArgsRest(__VLS_23));
const { default: __VLS_27 } = __VLS_25.slots;
const __VLS_28 = FaSelect;
// @ts-ignore
const __VLS_29 = __VLS_asFunctionalComponent1(__VLS_28, new __VLS_28({
    modelValue: (__VLS_ctx.currency),
    options: ([
        { label: '¥', value: 'CNY' },
        { label: '$', value: 'USD' },
        { label: '€', value: 'EUR' },
    ]),
    ...{ class: "gap-1 w-inherit" },
}));
const __VLS_30 = __VLS_29({
    modelValue: (__VLS_ctx.currency),
    options: ([
        { label: '¥', value: 'CNY' },
        { label: '$', value: 'USD' },
        { label: '€', value: 'EUR' },
    ]),
    ...{ class: "gap-1 w-inherit" },
}, ...__VLS_functionalComponentArgsRest(__VLS_29));
/** @type {__VLS_StyleScopedClasses['gap-1']} */ ;
/** @type {__VLS_StyleScopedClasses['w-inherit']} */ ;
const __VLS_33 = FaInput;
// @ts-ignore
const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({
    placeholder: "10.00",
}));
const __VLS_35 = __VLS_34({
    placeholder: "10.00",
}, ...__VLS_functionalComponentArgsRest(__VLS_34));
// @ts-ignore
[currency,];
var __VLS_25;
const __VLS_38 = FaButtonGroup || FaButtonGroup;
// @ts-ignore
const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({}));
const __VLS_40 = __VLS_39({}, ...__VLS_functionalComponentArgsRest(__VLS_39));
const { default: __VLS_43 } = __VLS_41.slots;
const __VLS_44 = FaButton || FaButton;
// @ts-ignore
const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
    variant: "outline",
}));
const __VLS_46 = __VLS_45({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_45));
const { default: __VLS_49 } = __VLS_47.slots;
// @ts-ignore
[];
var __VLS_47;
const __VLS_50 = FaDropdown || FaDropdown;
// @ts-ignore
const __VLS_51 = __VLS_asFunctionalComponent1(__VLS_50, new __VLS_50({
    items: ([
        [
            { label: '加入黑名单' },
            { label: '分享到群聊' },
            { label: '反馈举报' },
        ],
        [
            { label: '取消关注' },
        ],
    ]),
}));
const __VLS_52 = __VLS_51({
    items: ([
        [
            { label: '加入黑名单' },
            { label: '分享到群聊' },
            { label: '反馈举报' },
        ],
        [
            { label: '取消关注' },
        ],
    ]),
}, ...__VLS_functionalComponentArgsRest(__VLS_51));
const { default: __VLS_55 } = __VLS_53.slots;
const __VLS_56 = FaButton || FaButton;
// @ts-ignore
const __VLS_57 = __VLS_asFunctionalComponent1(__VLS_56, new __VLS_56({
    variant: "outline",
    size: "icon",
}));
const __VLS_58 = __VLS_57({
    variant: "outline",
    size: "icon",
}, ...__VLS_functionalComponentArgsRest(__VLS_57));
const { default: __VLS_61 } = __VLS_59.slots;
const __VLS_62 = FaIcon;
// @ts-ignore
const __VLS_63 = __VLS_asFunctionalComponent1(__VLS_62, new __VLS_62({
    name: "i-ep:caret-bottom",
}));
const __VLS_64 = __VLS_63({
    name: "i-ep:caret-bottom",
}, ...__VLS_functionalComponentArgsRest(__VLS_63));
// @ts-ignore
[];
var __VLS_59;
// @ts-ignore
[];
var __VLS_53;
// @ts-ignore
[];
var __VLS_41;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
