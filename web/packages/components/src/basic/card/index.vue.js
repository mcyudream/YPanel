import { Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle, } from './card';
defineOptions({
    name: 'BuiltInCard',
});
const props = defineProps();
const slots = defineSlots();
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Card | typeof __VLS_components.Card} */
Card;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ class: (props.class) },
}));
const __VLS_2 = __VLS_1({
    ...{ class: (props.class) },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
if (!!slots.header || !!__VLS_ctx.title || !!__VLS_ctx.description) {
    let __VLS_7;
    /** @ts-ignore @type { | typeof __VLS_components.CardHeader | typeof __VLS_components.CardHeader} */
    CardHeader;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
        ...{ class: (props.headerClass) },
    }));
    const __VLS_9 = __VLS_8({
        ...{ class: (props.headerClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_8));
    const { default: __VLS_12 } = __VLS_10.slots;
    __VLS_asFunctionalSlot(slots.header)({});
    if (!!__VLS_ctx.title) {
        let __VLS_14;
        /** @ts-ignore @type { | typeof __VLS_components.CardTitle | typeof __VLS_components.CardTitle} */
        CardTitle;
        // @ts-ignore
        const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({}));
        const __VLS_16 = __VLS_15({}, ...__VLS_functionalComponentArgsRest(__VLS_15));
        const { default: __VLS_19 } = __VLS_17.slots;
        (__VLS_ctx.title);
        // @ts-ignore
        [title, title, title, description,];
        var __VLS_17;
    }
    if (!!__VLS_ctx.description) {
        let __VLS_20;
        /** @ts-ignore @type { | typeof __VLS_components.CardDescription | typeof __VLS_components.CardDescription} */
        CardDescription;
        // @ts-ignore
        const __VLS_21 = __VLS_asFunctionalComponent1(__VLS_20, new __VLS_20({}));
        const __VLS_22 = __VLS_21({}, ...__VLS_functionalComponentArgsRest(__VLS_21));
        const { default: __VLS_25 } = __VLS_23.slots;
        (__VLS_ctx.description);
        // @ts-ignore
        [description, description,];
        var __VLS_23;
    }
    // @ts-ignore
    [];
    var __VLS_10;
}
if (!!slots.default) {
    let __VLS_26;
    /** @ts-ignore @type { | typeof __VLS_components.CardContent | typeof __VLS_components.CardContent} */
    CardContent;
    // @ts-ignore
    const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
        ...{ class: (props.contentClass) },
    }));
    const __VLS_28 = __VLS_27({
        ...{ class: (props.contentClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_27));
    const { default: __VLS_31 } = __VLS_29.slots;
    __VLS_asFunctionalSlot(slots['default'])({});
    // @ts-ignore
    [];
    var __VLS_29;
}
if (!!slots.footer) {
    let __VLS_33;
    /** @ts-ignore @type { | typeof __VLS_components.CardFooter | typeof __VLS_components.CardFooter} */
    CardFooter;
    // @ts-ignore
    const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({
        ...{ class: (props.footerClass) },
    }));
    const __VLS_35 = __VLS_34({
        ...{ class: (props.footerClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_34));
    const { default: __VLS_38 } = __VLS_36.slots;
    __VLS_asFunctionalSlot(slots.footer)({});
    // @ts-ignore
    [];
    var __VLS_36;
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeProps: {},
});
const __VLS_export = {};
export default {};
