import { reactive } from 'vue';
import FaButton from '../../button/index.vue';
import FaInput from '../../input/index.vue';
import FaSelect from '../../select/index.vue';
import FaSearchBar from '../index.vue';
const form = reactive({
    keyword: '',
    status: 'all',
    department: '',
    role: 'all',
    source: 'all',
    creator: '',
});
const statusOptions = [
    { label: '全部状态', value: 'all' },
    { label: '启用', value: 'enabled' },
    { label: '禁用', value: 'disabled' },
];
const roleOptions = [
    { label: '全部角色', value: 'all' },
    { label: '管理员', value: 'admin' },
    { label: '运营', value: 'operator' },
    { label: '访客', value: 'guest' },
];
const sourceOptions = [
    { label: '全部来源', value: 'all' },
    { label: '后台创建', value: 'admin' },
    { label: '用户注册', value: 'register' },
    { label: '批量导入', value: 'import' },
];
const __VLS_ctx = {
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
const __VLS_0 = FaSearchBar || FaSearchBar;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({}));
const __VLS_2 = __VLS_1({}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
{
    const { default: __VLS_7 } = __VLS_3.slots;
    const [{ fold }] = __VLS_vSlot(__VLS_7);
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "gap-3 grid grid-cols-1 md:grid-cols-[repeat(auto-fit,minmax(350px,1fr))]" },
    });
    /** @type {__VLS_StyleScopedClasses['gap-3']} */ ;
    /** @type {__VLS_StyleScopedClasses['grid']} */ ;
    /** @type {__VLS_StyleScopedClasses['grid-cols-1']} */ ;
    /** @type {__VLS_StyleScopedClasses['md:grid-cols-[repeat(auto-fit,minmax(350px,1fr))]']} */ ;
    const __VLS_8 = FaInput;
    // @ts-ignore
    const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
        modelValue: (__VLS_ctx.form.keyword),
        placeholder: "搜索用户名",
        ...{ class: "w-full" },
    }));
    const __VLS_10 = __VLS_9({
        modelValue: (__VLS_ctx.form.keyword),
        placeholder: "搜索用户名",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_9));
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    const __VLS_13 = FaSelect;
    // @ts-ignore
    const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
        modelValue: (__VLS_ctx.form.status),
        options: (__VLS_ctx.statusOptions),
        ...{ class: "w-full" },
    }));
    const __VLS_15 = __VLS_14({
        modelValue: (__VLS_ctx.form.status),
        options: (__VLS_ctx.statusOptions),
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_14));
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    const __VLS_18 = FaInput;
    // @ts-ignore
    const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
        modelValue: (__VLS_ctx.form.department),
        placeholder: "部门",
        ...{ class: "w-full" },
    }));
    const __VLS_20 = __VLS_19({
        modelValue: (__VLS_ctx.form.department),
        placeholder: "部门",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_19));
    __VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (!fold), }, null, null);
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    const __VLS_23 = FaSelect;
    // @ts-ignore
    const __VLS_24 = __VLS_asFunctionalComponent1(__VLS_23, new __VLS_23({
        modelValue: (__VLS_ctx.form.role),
        options: (__VLS_ctx.roleOptions),
        ...{ class: "w-full" },
    }));
    const __VLS_25 = __VLS_24({
        modelValue: (__VLS_ctx.form.role),
        options: (__VLS_ctx.roleOptions),
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_24));
    __VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (!fold), }, null, null);
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    const __VLS_28 = FaSelect;
    // @ts-ignore
    const __VLS_29 = __VLS_asFunctionalComponent1(__VLS_28, new __VLS_28({
        modelValue: (__VLS_ctx.form.source),
        options: (__VLS_ctx.sourceOptions),
        ...{ class: "w-full" },
    }));
    const __VLS_30 = __VLS_29({
        modelValue: (__VLS_ctx.form.source),
        options: (__VLS_ctx.sourceOptions),
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_29));
    __VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (!fold), }, null, null);
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    const __VLS_33 = FaInput;
    // @ts-ignore
    const __VLS_34 = __VLS_asFunctionalComponent1(__VLS_33, new __VLS_33({
        modelValue: (__VLS_ctx.form.creator),
        placeholder: "创建人",
        ...{ class: "w-full" },
    }));
    const __VLS_35 = __VLS_34({
        modelValue: (__VLS_ctx.form.creator),
        placeholder: "创建人",
        ...{ class: "w-full" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_34));
    __VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (!fold), }, null, null);
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "flex gap-2 col-end--1 justify-end" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['col-end--1']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-end']} */ ;
    const __VLS_38 = FaButton || FaButton;
    // @ts-ignore
    const __VLS_39 = __VLS_asFunctionalComponent1(__VLS_38, new __VLS_38({}));
    const __VLS_40 = __VLS_39({}, ...__VLS_functionalComponentArgsRest(__VLS_39));
    const { default: __VLS_43 } = __VLS_41.slots;
    // @ts-ignore
    [form, form, form, form, form, form, statusOptions, roleOptions, sourceOptions,];
    var __VLS_41;
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
    // @ts-ignore
    [];
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({});
export default {};
