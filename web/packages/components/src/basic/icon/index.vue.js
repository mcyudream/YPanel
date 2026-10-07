import { Icon } from '@iconify/vue';
import { UseImage } from '@vueuse/components';
import { computed, defineComponent, h, Transition } from 'vue';
import { cn } from '#utils';
defineOptions({
    name: 'BuiltInIcon',
});
const props = defineProps();
const outputType = computed(() => {
    if (!props.name) {
        return;
    }
    const hasPathFeatures = (str) => {
        return /^\.{1,2}\//.test(str) || str.startsWith('/') || str.includes('/');
    };
    if (/^https?:\/\//.test(props.name) || hasPathFeatures(props.name)) {
        return 'img';
    }
    else if (/i-[^:]+:[^:]+/.test(props.name)) {
        return 'unocss';
    }
    else if (props.name.startsWith('yd:')) {
        // YPanel 主通道图标（morphicons 本地渲染，见 YdMorphIcon）
        return 'yd';
    }
    else if (props.name.includes(':')) {
        return 'iconify';
    }
    else {
        return 'svg';
    }
});
const iconKey = computed(() => `${outputType.value ?? 'empty'}-${props.name}`);
// yd: 分支委托给全局注册的 YdMorphIcon（main.ts 注册）；未注册时静默回落空占位
const YdIconComponent = getCurrentInstance()?.appContext.components?.YdMorphIcon ?? null;
const IconWrapper = defineComponent({
    render() {
        const content = this.$slots.default?.();
        return props.transition
            ? h(Transition, { name: 'icon-switch' }, () => h('span', {
                key: iconKey.value,
                class: 'icon-switch-layer',
            }, content))
            : content;
    },
});
const __VLS_ctx = {
    ...{},
    ...{},
    ...{},
    ...{},
};
let __VLS_components;
let __VLS_intrinsics;
let __VLS_directives;
/** @type {__VLS_StyleScopedClasses['icon-switch-enter-from']} */ ;
/** @type {__VLS_StyleScopedClasses['icon-switch-leave-to']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.i, __VLS_intrinsics.i)({
    ...{ class: (__VLS_ctx.cn('relative size-[1em] flex-inline items-center justify-center fill-current leading-[1em]', props.class)) },
});
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.IconWrapper | typeof __VLS_components.IconWrapper} */
IconWrapper;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({}));
const __VLS_2 = __VLS_1({}, ...__VLS_functionalComponentArgsRest(__VLS_1));
const { default: __VLS_5 } = __VLS_3.slots;
if (__VLS_ctx.outputType === 'unocss') {
    __VLS_asFunctionalElement1(__VLS_intrinsics.i)({
        key: (`unocss-${__VLS_ctx.name}`),
        ...{ class: "shrink-0 size-inherit inset-0 absolute" },
        ...{ class: (__VLS_ctx.name) },
    });
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
}
else if (__VLS_ctx.outputType === 'yd' && __VLS_ctx.YdIconComponent) {
    const __VLS_6 = (__VLS_ctx.YdIconComponent);
    // @ts-ignore
    const __VLS_7 = __VLS_asFunctionalComponent1(__VLS_6, new __VLS_6({
        key: (`yd-${__VLS_ctx.name}`),
        name: (__VLS_ctx.name.slice(3)),
        ...{ class: "shrink-0 size-inherit!" },
    }));
    const __VLS_8 = __VLS_7({
        key: (`yd-${__VLS_ctx.name}`),
        name: (__VLS_ctx.name.slice(3)),
        ...{ class: "shrink-0 size-inherit!" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_7));
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit!']} */ ;
}
else if (__VLS_ctx.outputType === 'iconify') {
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        key: (`iconify-${__VLS_ctx.name}`),
        ...{ class: "flex shrink-0 size-inherit items-center inset-0 justify-center absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    let __VLS_11;
    /** @ts-ignore @type { | typeof __VLS_components.Icon} */
    Icon;
    // @ts-ignore
    const __VLS_12 = __VLS_asFunctionalComponent1(__VLS_11, new __VLS_11({
        icon: (__VLS_ctx.name),
        ...{ class: "shrink-0 size-inherit!" },
    }));
    const __VLS_13 = __VLS_12({
        icon: (__VLS_ctx.name),
        ...{ class: "shrink-0 size-inherit!" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_12));
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit!']} */ ;
}
else if (__VLS_ctx.outputType === 'svg') {
    __VLS_asFunctionalElement1(__VLS_intrinsics.svg, __VLS_intrinsics.svg)({
        key: (`svg-${__VLS_ctx.name}`),
        ...{ class: "shrink-0 size-inherit inset-0 absolute" },
        'aria-hidden': "true",
    });
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.use)({
        'xlink:href': (`./__spritemap#sprite-${__VLS_ctx.name}`),
    });
}
else if (__VLS_ctx.outputType === 'img') {
    __VLS_asFunctionalElement1(__VLS_intrinsics.span, __VLS_intrinsics.span)({
        key: (`img-${__VLS_ctx.name}`),
        ...{ class: "flex shrink-0 size-inherit items-center inset-0 justify-center absolute" },
    });
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    let __VLS_16;
    /** @ts-ignore @type { | typeof __VLS_components.UseImage | typeof __VLS_components.UseImage} */
    UseImage;
    // @ts-ignore
    const __VLS_17 = __VLS_asFunctionalComponent1(__VLS_16, new __VLS_16({
        src: (__VLS_ctx.name),
        ...{ class: "shrink-0 size-inherit" },
    }));
    const __VLS_18 = __VLS_17({
        src: (__VLS_ctx.name),
        ...{ class: "shrink-0 size-inherit" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_17));
    /** @type {__VLS_StyleScopedClasses['shrink-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
    const { default: __VLS_21 } = __VLS_19.slots;
    {
        const { loading: __VLS_22 } = __VLS_19.slots;
        __VLS_asFunctionalElement1(__VLS_intrinsics.i)({
            ...{ class: "i-line-md:loading-loop size-inherit" },
        });
        /** @type {__VLS_StyleScopedClasses['i-line-md:loading-loop']} */ ;
        /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
        // @ts-ignore
        [cn, outputType, outputType, outputType, outputType, outputType, name, name, name, name, name, name, name, name, name, name, YdIconComponent, YdIconComponent,];
    }
    {
        const { error: __VLS_23 } = __VLS_19.slots;
        __VLS_asFunctionalElement1(__VLS_intrinsics.i)({
            ...{ class: "i-ph:image-broken-duotone size-inherit" },
        });
        /** @type {__VLS_StyleScopedClasses['i-ph:image-broken-duotone']} */ ;
        /** @type {__VLS_StyleScopedClasses['size-inherit']} */ ;
        // @ts-ignore
        [];
    }
    // @ts-ignore
    [];
    var __VLS_19;
}
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeProps: {},
});
export default {};
