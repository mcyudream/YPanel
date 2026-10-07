import { useId } from 'reka-ui';
import { useField } from 'vee-validate';
import { cloneVNode, Comment, computed, defineComponent, Fragment, h, inject, isVNode, onBeforeUnmount, onMounted, provide, toRef, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import { FORM_LAYOUT_INJECTION_KEY } from './context';
import FormControl from './form/FormControl.vue';
import FormDescription from './form/FormDescription.vue';
import FormLabel from './form/FormLabel.vue';
import FormMessage from './form/FormMessage.vue';
import { FORM_ITEM_INJECTION_KEY } from './form/injectionKeys';
export default {};
const __VLS_export = ((__VLS_props, __VLS_ctx, __VLS_exposed, __VLS_setup = (async () => {
    defineOptions({
        name: 'BuiltInFormItem',
    });
    const props = withDefaults(defineProps(), {
        autoBind: true,
    });
    const slots = defineSlots();
    const injectedFormContext = inject(FORM_LAYOUT_INJECTION_KEY);
    if (!injectedFormContext) {
        throw new Error('FaFormItem should be used within <FaForm>.');
    }
    const formContext = injectedFormContext;
    const id = useId();
    provide(FORM_ITEM_INJECTION_KEY, id);
    const itemRef = useTemplateRef('itemRef');
    const field = useField(toRef(props, 'name'), toRef(props, 'rules'), {
        label: toRef(props, 'label'),
        validateOnValueUpdate: true,
    });
    const resolvedLabelPlacement = computed(() => props.labelPlacement ?? formContext.labelPlacement.value);
    const resolvedLabelWidth = computed(() => props.labelWidth ?? formContext.labelWidth.value);
    const resolvedDisabled = computed(() => formContext.disabled.value);
    const rootClass = computed(() => cn('min-w-0', resolvedLabelPlacement.value === 'top'
        ? 'grid gap-2'
        : 'grid gap-2 sm:grid-cols-[var(--fa-form-label-width)_minmax(0,1fr)] sm:items-start sm:gap-x-3', props.class));
    const rootStyle = computed(() => ({
        '--fa-form-label-width': resolveSize(resolvedLabelWidth.value),
    }));
    const labelClass = computed(() => cn('text-sm font-medium leading-none', resolvedLabelPlacement.value === 'top'
        ? 'inline-flex items-center gap-1 px-1'
        : 'flex min-h-9 items-center gap-1 sm:pt-0', resolvedLabelPlacement.value === 'right' && 'sm:justify-end sm:text-right', resolvedLabelPlacement.value === 'left' && 'sm:justify-start sm:text-left', props.labelClass));
    const contentClass = computed(() => cn('grid min-w-0 gap-1', resolvedLabelPlacement.value !== 'top' && 'sm:col-start-2', props.contentClass));
    const hasMessage = computed(() => Boolean(field.errorMessage.value));
    const hasDescription = computed(() => Boolean(props.description || slots.description));
    const feedbackClass = computed(() => cn('grid min-h-4 px-1'));
    const messageClass = computed(() => cn('col-start-1 row-start-1 text-xs', props.messageClass));
    const descriptionClass = computed(() => cn('col-start-1 row-start-1 text-xs', hasMessage.value && 'invisible', props.descriptionClass));
    function handleUpdateModelValue(value) {
        field.handleChange(value, true);
    }
    function handleBlur(event) {
        field.handleBlur(event, true);
    }
    function handleInput(event) {
        field.handleChange(event, true);
    }
    function handleChange(event) {
        field.handleChange(event, true);
    }
    const componentField = computed(() => ({
        'name': props.name,
        'modelValue': field.value.value,
        'onBlur': handleBlur,
        'onInput': handleInput,
        'onChange': handleChange,
        'onUpdate:modelValue': handleUpdateModelValue,
    }));
    const slotProps = computed(() => ({
        componentField: componentField.value,
    }));
    const ControlSlot = defineComponent({
        name: 'BuiltInFormItemControlSlot',
        setup() {
            return () => {
                const nodes = normalizeSlotNodes(slots.default?.(slotProps.value) ?? []);
                if (!nodes.length) {
                    return null;
                }
                const [firstNode, ...restNodes] = nodes;
                const controlNode = cloneVNode(firstNode, resolveControlProps(firstNode));
                return h(FormControl, null, {
                    default: () => [controlNode, ...restNodes],
                });
            };
        },
    });
    function resolveSize(size) {
        if (size == null || size === '') {
            return '96px';
        }
        return typeof size === 'number' ? `${size}px` : size;
    }
    function normalizeSlotNodes(nodes) {
        const result = [];
        const source = Array.isArray(nodes) ? nodes : [nodes];
        source.forEach((node) => {
            if (Array.isArray(node)) {
                result.push(...normalizeSlotNodes(node));
                return;
            }
            if (!isVNode(node) || node.type === Comment) {
                return;
            }
            if (node.type === Fragment && Array.isArray(node.children)) {
                result.push(...normalizeSlotNodes(node.children));
                return;
            }
            result.push(node);
        });
        return result;
    }
    function resolveControlProps(vnode) {
        if (!props.autoBind) {
            return {};
        }
        const vnodeProps = vnode.props ?? {};
        const nextProps = {
            'name': props.name,
            'disabled': resolvedDisabled.value || Boolean(vnodeProps.disabled),
            'aria-required': props.required ? 'true' : undefined,
            'onBlur': chainHandlers(vnodeProps.onBlur, componentField.value.onBlur),
            'onInput': chainHandlers(vnodeProps.onInput, componentField.value.onInput),
            'onChange': chainHandlers(vnodeProps.onChange, componentField.value.onChange),
            'onUpdate:modelValue': chainHandlers(vnodeProps['onUpdate:modelValue'], componentField.value['onUpdate:modelValue']),
        };
        if (!Object.hasOwn(vnodeProps, 'modelValue')) {
            nextProps.modelValue = componentField.value.modelValue;
        }
        return nextProps;
    }
    function chainHandlers(existing, next) {
        if (!existing || existing === next) {
            return next;
        }
        return (...args) => {
            next(...args);
            callHandler(existing, ...args);
        };
    }
    function callHandler(handler, ...args) {
        if (Array.isArray(handler)) {
            handler.forEach(item => item(...args));
            return;
        }
        handler(...args);
    }
    function registerCurrentField() {
        if (itemRef.value) {
            formContext.registerField(props.name, itemRef.value);
        }
    }
    function unregisterCurrentField(name = props.name) {
        formContext.unregisterField(name, itemRef.value ?? undefined);
    }
    onMounted(() => {
        registerCurrentField();
    });
    watch(() => props.name, (name, oldName) => {
        if (oldName) {
            unregisterCurrentField(oldName);
        }
        if (name) {
            registerCurrentField();
        }
    });
    onBeforeUnmount(() => {
        unregisterCurrentField();
    });
    const __VLS_defaults = {
        autoBind: true,
    };
    const __VLS_ctx = {
        ...{},
        ...{},
        ...{},
        ...{},
    };
    let __VLS_components;
    let __VLS_intrinsics;
    let __VLS_directives;
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ref: "itemRef",
        'data-slot': "form-item",
        ...{ class: (__VLS_ctx.rootClass) },
        ...{ style: (__VLS_ctx.rootStyle) },
    });
    if (props.label || __VLS_ctx.$slots.label) {
        const __VLS_0 = FormLabel || FormLabel;
        // @ts-ignore
        const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
            ...{ class: (__VLS_ctx.labelClass) },
        }));
        const __VLS_2 = __VLS_1({
            ...{ class: (__VLS_ctx.labelClass) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_1));
        const { default: __VLS_5 } = __VLS_3.slots;
        __VLS_asFunctionalSlot(slots.label)({
            ...(__VLS_ctx.slotProps),
        });
        __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({});
        (props.label);
        if (props.required) {
            __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
                ...{ class: "text-destructive" },
                'aria-hidden': "true",
            });
            /** @type {__VLS_StyleScopedClasses['text-destructive']} */ ;
        }
        // @ts-ignore
        [rootClass, rootStyle, $slots, labelClass, slotProps,];
        var __VLS_3;
    }
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: (__VLS_ctx.contentClass) },
    });
    let __VLS_7;
    /** @ts-ignore @type { | typeof __VLS_components.ControlSlot} */
    ControlSlot;
    // @ts-ignore
    const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({}));
    const __VLS_9 = __VLS_8({}, ...__VLS_functionalComponentArgsRest(__VLS_8));
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: (__VLS_ctx.feedbackClass) },
    });
    if (__VLS_ctx.hasDescription) {
        const __VLS_12 = FormDescription || FormDescription;
        // @ts-ignore
        const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
            ...{ class: (__VLS_ctx.descriptionClass) },
            'aria-hidden': (__VLS_ctx.hasMessage ? 'true' : undefined),
        }));
        const __VLS_14 = __VLS_13({
            ...{ class: (__VLS_ctx.descriptionClass) },
            'aria-hidden': (__VLS_ctx.hasMessage ? 'true' : undefined),
        }, ...__VLS_functionalComponentArgsRest(__VLS_13));
        const { default: __VLS_17 } = __VLS_15.slots;
        __VLS_asFunctionalSlot(slots.description)({
            ...(__VLS_ctx.slotProps),
        });
        (props.description);
        // @ts-ignore
        [slotProps, contentClass, feedbackClass, hasDescription, descriptionClass, hasMessage,];
        var __VLS_15;
    }
    if (__VLS_ctx.hasMessage) {
        const __VLS_19 = FormMessage || FormMessage;
        // @ts-ignore
        const __VLS_20 = __VLS_asFunctionalComponent1(__VLS_19, new __VLS_19({
            ...{ class: (__VLS_ctx.messageClass) },
        }));
        const __VLS_21 = __VLS_20({
            ...{ class: (__VLS_ctx.messageClass) },
        }, ...__VLS_functionalComponentArgsRest(__VLS_20));
        const { default: __VLS_24 } = __VLS_22.slots;
        {
            const { default: __VLS_25 } = __VLS_22.slots;
            const [{ message }] = __VLS_vSlot(__VLS_25);
            __VLS_asFunctionalSlot(slots.message)({
                ...(__VLS_ctx.slotProps),
            });
            (message);
            // @ts-ignore
            [slotProps, hasMessage, messageClass,];
        }
        // @ts-ignore
        [];
        var __VLS_22;
    }
    // @ts-ignore
    [];
    return {};
})()) => ({}));
