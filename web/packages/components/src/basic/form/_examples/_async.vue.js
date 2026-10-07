import { ref, shallowRef } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaFormItem from '../FormItem.vue';
import FaForm from '../index.vue';
const submitted = shallowRef('');
const model = ref({
    username: '',
    email: '',
});
const unavailableNames = ['admin', 'root', 'fantastic'];
const validationSchema = {
    async username(value) {
        if (!value) {
            return '请输入用户名';
        }
        if (value.length < 3) {
            return '用户名至少 3 个字符';
        }
        const available = await checkUsernameAvailable(value);
        return available ? true : '该用户名已被占用';
    },
    email(value) {
        return /^\S[^\s@]*@\S[^\s.]*\.\S+$/.test(value) ? true : '请输入有效邮箱';
    },
};
async function checkUsernameAvailable(value) {
    await wait(2000);
    return !unavailableNames.includes(value.trim().toLowerCase());
}
function wait(ms) {
    return new Promise(resolve => window.setTimeout(resolve, ms));
}
function handleSubmit(values) {
    submitted.value = JSON.stringify(values, null, 2);
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
{
    const { default: __VLS_7 } = __VLS_3.slots;
    const [{ isSubmitting }] = __VLS_vSlot(__VLS_7);
    const __VLS_8 = FaFormItem || FaFormItem;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        name: "username",
        label: "用户名",
        required: true,
        description: "输入 admin / root / fantastic 会返回占用错误",
    }));
    const __VLS_10 = __VLS_9({
        name: "username",
        label: "用户名",
        required: true,
        description: "输入 admin / root / fantastic 会返回占用错误",
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    const { default: __VLS_13 } = __VLS_11.slots;
    const __VLS_14 = FaInput;
    // @ts-ignore
    const __VLS_15 = __VLS_asFunctionalComponent1(__VLS_14, new __VLS_14({
        placeholder: "请输入用户名",
        ...{ class: "w-full" },
    }));
    const __VLS_16 = __VLS_15({
        placeholder: "请输入用户名",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_15));
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    // @ts-ignore
    [model, validationSchema, handleSubmit,];
    var __VLS_11;
    const __VLS_19 = FaFormItem || FaFormItem;
    // @ts-ignore
    const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
        name: "email",
        label: "邮箱",
        required: true,
    }));
    const __VLS_21 = __VLS_20({
        name: "email",
        label: "邮箱",
        required: true,
    }, ...__VLS_functionalComponentArgsRest(__VLS_20));
    const { default: __VLS_24 } = __VLS_22.slots;
    const __VLS_25 = FaInput;
    // @ts-ignore
    const __VLS_26 = __VLS_asFunctionalComponent1(__VLS_25, new __VLS_25({
        type: "email",
        placeholder: "name@example.com",
        ...{ class: "w-full" },
    }));
    const __VLS_27 = __VLS_26({
        type: "email",
        placeholder: "name@example.com",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_26));
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    // @ts-ignore
    [];
    var __VLS_22;
    const __VLS_30 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({
        type: "submit",
        loading: (isSubmitting),
    }));
    const __VLS_32 = __VLS_31({
        type: "submit",
        loading: (isSubmitting),
    }, ...__VLS_functionalComponentArgsRest(__VLS_31));
    const { default: __VLS_35 } = __VLS_33.slots;
    // @ts-ignore
    [];
    var __VLS_33;
    // @ts-ignore
    [];
    __VLS_3.slots['' /* empty slot name completion */];
}
var __VLS_3;
var __VLS_4;
if (__VLS_ctx.submitted) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.pre, __VLS_intrinsics.pre)({
        ...{ class: "text-sm m-0 p-4 rounded-md bg-muted" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['m-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['p-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-md']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-muted']} */ ;
    (__VLS_ctx.submitted);
}
// @ts-ignore
[submitted, submitted,];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
