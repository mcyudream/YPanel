import { cloneDeep, isEqual, isPlainObject } from 'es-toolkit';
import { useForm } from 'vee-validate';
import { computed, nextTick, provide, toRaw, toRef, watch } from 'vue';
import { cn } from '#utils';
import { FORM_LAYOUT_INJECTION_KEY } from './context';
export default {};
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInForm',
    });
    const props = withDefaults(defineProps(), {
        as: 'form',
        labelPlacement: 'top',
    });
    const emit = defineEmits();
    const fieldElements = new Map();
    const form = useForm({
        validationSchema: toRef(props, 'validationSchema'),
        initialValues: cloneFormValue(props.model ?? props.initialValues),
        initialErrors: props.initialErrors,
        initialTouched: props.initialTouched,
        validateOnMount: props.validateOnMount,
        keepValuesOnUnmount: props.keepValuesOnUnmount,
        name: props.name,
    });
    let syncingFromModel = false;
    let syncingToModel = false;
    const resolvedLabelPlacement = computed(() => props.labelPlacement);
    const resolvedLabelWidth = computed(() => props.labelWidth ?? '96px');
    const resolvedDisabled = computed(() => Boolean(props.disabled));
    const rootClass = computed(() => cn('grid gap-1', props.class));
    const slotProps = computed(() => ({
        values: form.values,
        errors: form.errors.value,
        meta: form.meta.value,
        isSubmitting: form.isSubmitting.value,
        submitCount: form.submitCount.value,
        submit,
        validate,
        resetFields,
        clearValidate,
        scrollToField,
    }));
    function registerField(name, el) {
        fieldElements.set(name, el);
    }
    function unregisterField(name, el) {
        if (!el || fieldElements.get(name) === el) {
            fieldElements.delete(name);
        }
    }
    watch(() => props.model, (model) => {
        if (!model || syncingToModel) {
            return;
        }
        const nextValues = cloneFormValue(model);
        if (isEqual(toRaw(form.values), nextValues)) {
            return;
        }
        syncingFromModel = true;
        form.setValues(nextValues, false);
        void nextTick(() => {
            syncingFromModel = false;
        });
    }, { deep: true });
    watch(form.values, (values) => {
        if (!props.model || syncingFromModel) {
            return;
        }
        const nextModel = cloneFormValue(values);
        if (isEqual(toRaw(props.model), nextModel)) {
            return;
        }
        syncingToModel = true;
        syncModel(props.model, nextModel);
        emit('update:model', nextModel);
        void nextTick(() => {
            syncingToModel = false;
        });
    }, { deep: true });
    provide(FORM_LAYOUT_INJECTION_KEY, {
        labelPlacement: resolvedLabelPlacement,
        labelWidth: resolvedLabelWidth,
        disabled: resolvedDisabled,
        registerField,
        unregisterField,
    });
    function validate() {
        return form.validate();
    }
    function resetFields(state, opts) {
        form.resetForm(state, opts);
    }
    function clearValidate(name) {
        if (name) {
            form.setFieldError(name, undefined);
            return;
        }
        form.setErrors({});
    }
    function setFieldValue(name, value, shouldValidate = true) {
        form.setFieldValue(name, value, shouldValidate);
    }
    function setFieldError(name, message) {
        form.setFieldError(name, message);
    }
    function cloneFormValue(value) {
        return cloneDeep(toRaw(value));
    }
    function syncModel(target, source) {
        Object.keys(target).forEach((key) => {
            if (!Object.hasOwn(source, key)) {
                delete target[key];
            }
        });
        Object.entries(source).forEach(([key, value]) => {
            const currentValue = target[key];
            const rawCurrentValue = toRaw(currentValue);
            if (isPlainObject(rawCurrentValue) && isPlainObject(value)) {
                syncModel(currentValue, value);
                return;
            }
            if (!isEqual(rawCurrentValue, value)) {
                target[key] = value;
            }
        });
    }
    function scrollToField(name) {
        const el = fieldElements.get(String(name));
        if (!el) {
            return false;
        }
        el.scrollIntoView({
            block: 'center',
            behavior: 'smooth',
        });
        focusFieldControl(el);
        return true;
    }
    function focusFieldControl(el) {
        const focusable = el.querySelector('[data-slot="form-control"], input, textarea, select, button, [tabindex]:not([tabindex="-1"])');
        focusable?.focus?.({ preventScroll: true });
    }
    async function handleInvalidSubmit(ctx) {
        emit('invalidSubmit', ctx);
        if (!props.scrollToError) {
            return;
        }
        await nextTick();
        const [firstFieldName] = Object.keys(ctx.errors);
        if (firstFieldName) {
            scrollToField(firstFieldName);
        }
    }
    const submitForm = form.handleSubmit((values, ctx) => emit('submit', values, ctx), ctx => void handleInvalidSubmit(ctx));
    function submit() {
        return submitForm();
    }
    let __VLS_exposed;
    defineExpose({
        submit,
        validate,
        resetFields,
        clearValidate,
        setFieldValue,
        setFieldError,
        scrollToField,
    });
    const __VLS_defaults = {
        as: 'form',
        labelPlacement: 'top',
    };
    const __VLS_ctx = {
        ...{},
        ...{},
        ...{},
        ...{},
        ...{},
    };
    let __VLS_components;
    let __VLS_intrinsics;
    let __VLS_directives;
    const __VLS_0 = (props.as);
    // @ts-ignore
    const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
        ...{ 'onSubmit': {} },
        dataSlot: "form",
        ...{ class: (__VLS_ctx.rootClass) },
    }));
    const __VLS_2 = __VLS_1({
        ...{ 'onSubmit': {} },
        dataSlot: "form",
        ...{ class: (__VLS_ctx.rootClass) },
    }, ...__VLS_functionalComponentArgsRest(__VLS_1));
    let __VLS_5;
    const __VLS_6 = {
        /** @type {typeof __VLS_5.submit} */
        onSubmit: (__VLS_ctx.submitForm),
    };
    var __VLS_7;
    const { default: __VLS_8 } = __VLS_3.slots;
    var __VLS_9 = {
        ...(__VLS_ctx.slotProps),
    };
    // @ts-ignore
    [rootClass, submitForm, slotProps,];
    var __VLS_3;
    var __VLS_4;
    // @ts-ignore
    var __VLS_10 = __VLS_9;
    // @ts-ignore
    [];
    return {};
})()) => ({}));
