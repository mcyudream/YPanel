import { ref, shallowRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaIcon from '../../icon/index.vue';
import FaInput from '../../input/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const result = shallowRef('');
const inputAuth = shallowRef('');
const model = ref({
    auths: [],
});
const validationSchema = {
    auths(value) {
        return value.length ? true : '请至少添加一个权限标识';
    },
};
function addAuth() {
    const auth = inputAuth.value.trim();
    if (!auth) {
        return;
    }
    if (model.value.auths.includes(auth)) {
        inputAuth.value = '';
        return;
    }
    model.value.auths = [...model.value.auths, auth];
    inputAuth.value = '';
}
function removeAuth(auth) {
    model.value.auths = model.value.auths.filter(item => item !== auth);
}
function handleSubmit(values) {
    result.value = JSON.stringify(values, null, 2);
}
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "gap-4 grid max-w-160" },
});
/** @type {__VLS_StyleScopedClasses['gap-4']} */ ;
/** @type {__VLS_StyleScopedClasses['grid']} */ ;
/** @type {__VLS_StyleScopedClasses['max-w-160']} */ ;
const __VLS_0 = FaForm || FaForm;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onSubmit': {} },
    model: (__VLS_ctx.model),
    validationSchema: (__VLS_ctx.validationSchema),
}));
const __VLS_2 = __VLS_1({
    ...{ 'onSubmit': {} },
    model: (__VLS_ctx.model),
    validationSchema: (__VLS_ctx.validationSchema),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.submit} */
    onSubmit: (__VLS_ctx.handleSubmit),
};
const { default: __VLS_7 } = __VLS_3.slots;
const __VLS_8 = FaFormItem || FaFormItem;
// @ts-ignore
const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
    name: "auths",
    label: "权限标识",
    required: true,
    autoBind: (false),
}));
const __VLS_10 = __VLS_9({
    name: "auths",
    label: "权限标识",
    required: true,
    autoBind: (false),
}, ...__VLS_functionalComponentArgsRest(__VLS_9));
const { default: __VLS_13 } = __VLS_11.slots;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "flex flex-wrap gap-2 max-w-full items-center" },
});
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
/** @type {__VLS_StyleScopedClasses['max-w-full']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
for (const [item] of __VLS_vFor((__VLS_ctx.model.auths))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        key: (item),
        ...{ class: "text-xs text-secondary-foreground px-2.5 py-2 border rounded-lg bg-secondary inline-flex gap-1.5 max-w-full items-center" },
    });
    /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-secondary-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['px-2.5']} */ ;
    /** @type {__VLS_StyleScopedClasses['py-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['border']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-secondary']} */ ;
    /** @type {__VLS_StyleScopedClasses['inline-flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-1.5']} */ ;
    /** @type {__VLS_StyleScopedClasses['max-w-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        ...{ class: "truncate" },
        title: (item),
    });
    /** @type {__VLS_StyleScopedClasses['truncate']} */ ;
    (item);
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (...[$event]) => {
                return (__VLS_ctx.removeAuth(item));
                // @ts-ignore
                [model, model, validationSchema, handleSubmit, removeAuth,];
            } },
        type: "button",
        ...{ class: "flex-center size-4" },
    });
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-4']} */ ;
    const __VLS_14 = FaIcon;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        name: "i-ep:close",
    }));
    const __VLS_16 = __VLS_15({
        name: "i-ep:close",
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    // @ts-ignore
    [];
}
const __VLS_19 = FaInput;
// @ts-ignore
const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
    ...{ 'onKeyup': {} },
    modelValue: (__VLS_ctx.inputAuth),
    placeholder: "请输入权限标识",
    ...{ class: "w-50" },
}));
const __VLS_21 = __VLS_20({
    ...{ 'onKeyup': {} },
    modelValue: (__VLS_ctx.inputAuth),
    placeholder: "请输入权限标识",
    ...{ class: "w-50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_20));
let __VLS_24;
const __VLS_25 = {
    /** @type {typeof __VLS_24.keyup} */
    onKeyup: (__VLS_ctx.addAuth),
};
/** @type {__VLS_StyleScopedClasses['w-50']} */ ;
var __VLS_22;
var __VLS_23;
const __VLS_26 = FaButton || FaButton;
// @ts-ignore
const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}));
const __VLS_28 = __VLS_27({
    ...{ 'onClick': {} },
    type: "button",
    variant: "outline",
}, ...__VLS_functionalComponentArgsRest(__VLS_27));
let __VLS_31;
const __VLS_32 = {
    /** @type {typeof __VLS_31.click} */
    onClick: (__VLS_ctx.addAuth),
};
const { default: __VLS_33 } = __VLS_29.slots;
// @ts-ignore
[inputAuth, addAuth, addAuth,];
var __VLS_29;
var __VLS_30;
// @ts-ignore
[];
var __VLS_11;
const __VLS_34 = FaButton || FaButton;
// @ts-ignore
const __VLS_35 = __VLS_asFunctionalComponent1(__VLS_34, new __VLS_34({
    type: "submit",
}));
const __VLS_36 = __VLS_35({
    type: "submit",
}, ...__VLS_functionalComponentArgsRest(__VLS_35));
const { default: __VLS_39 } = __VLS_37.slots;
// @ts-ignore
[];
var __VLS_37;
// @ts-ignore
[];
var __VLS_3;
var __VLS_4;
if (__VLS_ctx.result) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.pre, __VLS_intrinsics.pre)({
        ...{ class: "text-sm m-0 p-4 rounded-md bg-muted" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['m-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['p-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-muted']} */ ;
    (__VLS_ctx.result);
}
// @ts-ignore
[result, result,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
