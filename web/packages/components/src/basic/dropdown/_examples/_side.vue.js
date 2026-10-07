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
        { label: 'Preview', icon: 'i-lucide:eye', handle: () => handleClick('Preview') },
        { label: 'Duplicate', icon: 'i-lucide:copy', handle: () => handleClick('Duplicate') },
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
    side: "top",
}));
const __VLS_2 = __VLS_1({
    items: (__VLS_ctx.items),
    side: "top",
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
    name: "i-ep:caret-top",
}));
const __VLS_14 = __VLS_13({
    name: "i-ep:caret-top",
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
    side: "right",
}));
const __VLS_19 = __VLS_18({
    items: (__VLS_ctx.items),
    side: "right",
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
    name: "i-ep:caret-right",
}));
const __VLS_31 = __VLS_30({
    name: "i-ep:caret-right",
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
    side: "bottom",
}));
const __VLS_36 = __VLS_35({
    items: (__VLS_ctx.items),
    side: "bottom",
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
const __VLS_51 = FaDropdown || FaDropdown;
// @ts-ignore
const __VLS_52 = __VLS_asFunctionalComponent1(__VLS_51, new __VLS_51({
    items: (__VLS_ctx.items),
    side: "left",
}));
const __VLS_53 = __VLS_52({
    items: (__VLS_ctx.items),
    side: "left",
}, ...__VLS_functionalComponentArgsRest(__VLS_52));
const { default: __VLS_56 } = __VLS_54.slots;
const __VLS_57 = FaButton || FaButton;
// @ts-ignore
const __VLS_58 = __VLS_asFunctionalComponent1(__VLS_57, new __VLS_57({
    variant: "outline",
}));
const __VLS_59 = __VLS_58({
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_58));
const { default: __VLS_62 } = __VLS_60.slots;
const __VLS_63 = FaIcon;
// @ts-ignore
const __VLS_64 = __VLS_asFunctionalComponent1(__VLS_63, new __VLS_63({
    name: "i-ep:caret-left",
}));
const __VLS_65 = __VLS_64({
    name: "i-ep:caret-left",
}, ...__VLS_functionalComponentArgsRest(__VLS_64));
// @ts-ignore
[items,];
var __VLS_60;
// @ts-ignore
[];
var __VLS_54;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
