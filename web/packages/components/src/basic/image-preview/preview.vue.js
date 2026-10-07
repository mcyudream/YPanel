import { VisuallyHidden } from 'reka-ui';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import ButtonGroup from '../button/ButtonGroup.vue';
import Button from '../button/index.vue';
import Icon from '../icon/index.vue';
import { Dialog, DialogContent, DialogDescription, DialogTitle } from './dialog';
const props = defineProps();
const isOpen = defineModel();
// 多图操作
const index = ref(props.index ?? 0);
const srcList = computed(() => {
    if (Array.isArray(props.src)) {
        return props.src;
    }
    return [props.src];
});
function handlePrev() {
    index.value = index.value - 1;
    if (index.value < 0) {
        index.value = srcList.value.length - 1;
    }
}
function handleNext() {
    index.value = index.value + 1;
    if (index.value >= srcList.value.length) {
        index.value = 0;
    }
}
watch(index, resetImageState);
// 图片操作相关状态
const scale = ref(1);
const rotate = ref(0);
const position = ref({ x: 0, y: 0 });
const isDragging = ref(false);
const dragStart = ref({ x: 0, y: 0 });
// 图片操作函数
function handleZoomIn() {
    scale.value = Math.min(scale.value + 0.25, 3);
}
function handleZoomOut() {
    scale.value = Math.max(scale.value - 0.25, 0.5);
}
function handleOriginalSize() {
    scale.value = 1;
}
function handleRotateLeft() {
    rotate.value = rotate.value - 90;
}
function handleRotateRight() {
    rotate.value = rotate.value + 90;
}
// 处理滚轮缩放
function handleWheel(e) {
    e.preventDefault();
    if (e.deltaY < 0) {
        handleZoomIn();
    }
    else {
        handleZoomOut();
    }
}
// 处理鼠标按下事件
function handleMouseDown(e) {
    isDragging.value = true;
    dragStart.value = {
        x: e.clientX - position.value.x,
        y: e.clientY - position.value.y,
    };
}
// 处理鼠标移动事件
function handleMouseMove(e) {
    if (!isDragging.value) {
        return;
    }
    position.value = {
        x: e.clientX - dragStart.value.x,
        y: e.clientY - dragStart.value.y,
    };
}
// 处理鼠标松开事件
function handleMouseUp() {
    isDragging.value = false;
}
// 重置图片状态
function resetImageState() {
    scale.value = 1;
    rotate.value = 0;
    position.value = { x: 0, y: 0 };
}
// 监听鼠标事件
onMounted(() => {
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
});
onUnmounted(() => {
    window.removeEventListener('mousemove', handleMouseMove);
    window.removeEventListener('mouseup', handleMouseUp);
});
function handleAnimationEnd() {
    if (!isOpen.value) {
        resetImageState();
    }
}
let __VLS_modelEmit;
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
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.Dialog | typeof __VLS_components.Dialog} */
Dialog;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    open: (__VLS_ctx.isOpen),
}));
const __VLS_2 = __VLS_1({
    open: (__VLS_ctx.isOpen),
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
var __VLS_5;
const { default: __VLS_6 } = __VLS_3.slots;
let __VLS_7;
/** @ts-ignore @type { | typeof __VLS_components.DialogContent | typeof __VLS_components.DialogContent} */
DialogContent;
// @ts-ignore
const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
    ...{ 'onAnimationEnd': {} },
}));
const __VLS_9 = __VLS_8({
    ...{ 'onAnimationEnd': {} },
}, ...__VLS_functionalComponentArgsRest(__VLS_8));
let __VLS_12;
const __VLS_13 = {
    /** @type {typeof __VLS_12.animationEnd} */
    onAnimationEnd: (__VLS_ctx.handleAnimationEnd),
};
const { default: __VLS_14 } = __VLS_10.slots;
let __VLS_15;
/** @ts-ignore @type { | typeof __VLS_components.VisuallyHidden | typeof __VLS_components.VisuallyHidden} */
VisuallyHidden;
// @ts-ignore
const __VLS_16 = __VLS_asFunctionalComponent1(__VLS_15, new __VLS_15({}));
const __VLS_17 = __VLS_16({}, ...__VLS_functionalComponentArgsRest(__VLS_16));
const { default: __VLS_20 } = __VLS_18.slots;
let __VLS_21;
/** @ts-ignore @type { | typeof __VLS_components.DialogTitle} */
DialogTitle;
// @ts-ignore
const __VLS_22 = __VLS_asFunctionalComponent1(__VLS_21, new __VLS_21({}));
const __VLS_23 = __VLS_22({}, ...__VLS_functionalComponentArgsRest(__VLS_22));
let __VLS_26;
/** @ts-ignore @type { | typeof __VLS_components.DialogDescription} */
DialogDescription;
// @ts-ignore
const __VLS_27 = __VLS_asFunctionalComponent1(__VLS_26, new __VLS_26({}));
const __VLS_28 = __VLS_27({}, ...__VLS_functionalComponentArgsRest(__VLS_27));
// @ts-ignore
[isOpen, handleAnimationEnd,];
var __VLS_18;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ onWheel: (__VLS_ctx.handleWheel) },
    ...{ class: "flex-center size-full relative" },
});
/** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
/** @type {__VLS_StyleScopedClasses['size-full']} */ ;
/** @type {__VLS_StyleScopedClasses['relative']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.img)({
    ...{ onMousedown: (__VLS_ctx.handleMouseDown) },
    src: (__VLS_ctx.srcList[__VLS_ctx.index]),
    ...{ class: "mx-auto max-h-full max-w-full object-contain" },
    ...{ class: ({
            'transition-all duration-300': !__VLS_ctx.isDragging,
        }) },
    ...{ style: ({
            transform: `translate(${__VLS_ctx.position.x}px, ${__VLS_ctx.position.y}px) scale(${__VLS_ctx.scale}) rotate(${__VLS_ctx.rotate}deg)`,
            cursor: __VLS_ctx.isDragging ? 'grabbing' : 'grab',
        }) },
});
/** @type {__VLS_StyleScopedClasses['mx-auto']} */ ;
/** @type {__VLS_StyleScopedClasses['max-h-full']} */ ;
/** @type {__VLS_StyleScopedClasses['max-w-full']} */ ;
/** @type {__VLS_StyleScopedClasses['object-contain']} */ ;
/** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
/** @type {__VLS_StyleScopedClasses['duration-300']} */ ;
if (__VLS_ctx.srcList.length > 1) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-sm text-muted-foreground bottom-20 left-1/2 absolute -translate-x-1/2" },
    });
    /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-muted-foreground']} */ ;
    /** @type {__VLS_StyleScopedClasses['bottom-20']} */ ;
    /** @type {__VLS_StyleScopedClasses['left-1/2']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    /** @type {__VLS_StyleScopedClasses['-translate-x-1/2']} */ ;
    (__VLS_ctx.index + 1);
    (__VLS_ctx.srcList.length);
    const __VLS_31 = Button || Button;
    // @ts-ignore
    const __VLS_32 = __VLS_asFunctionalComponent1(__VLS_31, new __VLS_31({
        ...{ 'onClick': {} },
        variant: "ghost",
        size: "icon",
        ...{ class: "rounded-full bg-muted/50 scale-125 left-4 top-1/2 absolute backdrop-blur-xs -translate-y-1/2" },
    }));
    const __VLS_33 = __VLS_32({
        ...{ 'onClick': {} },
        variant: "ghost",
        size: "icon",
        ...{ class: "rounded-full bg-muted/50 scale-125 left-4 top-1/2 absolute backdrop-blur-xs -translate-y-1/2" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_32));
    let __VLS_36;
    const __VLS_37 = {
        /** @type {typeof __VLS_36.click} */
        onClick: (__VLS_ctx.handlePrev),
    };
    /** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
    /** @type {__VLS_StyleScopedClasses['scale-125']} */ ;
    /** @type {__VLS_StyleScopedClasses['left-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-1/2']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    /** @type {__VLS_StyleScopedClasses['backdrop-blur-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['-translate-y-1/2']} */ ;
    const { default: __VLS_38 } = __VLS_34.slots;
    const __VLS_39 = Icon;
    // @ts-ignore
    const __VLS_40 = __VLS_asFunctionalComponent1(__VLS_39, new __VLS_39({
        name: "i-lucide:chevron-left",
        ...{ class: "size-6" },
    }));
    const __VLS_41 = __VLS_40({
        name: "i-lucide:chevron-left",
        ...{ class: "size-6" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_40));
    /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
    // @ts-ignore
    [handleWheel, handleMouseDown, srcList, srcList, srcList, index, index, isDragging, isDragging, position, position, scale, rotate, handlePrev,];
    var __VLS_34;
    var __VLS_35;
    const __VLS_44 = Button || Button;
    // @ts-ignore
    const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
        ...{ 'onClick': {} },
        variant: "ghost",
        size: "icon",
        ...{ class: "rounded-full bg-muted/50 scale-125 right-4 top-1/2 absolute backdrop-blur-xs -translate-y-1/2" },
    }));
    const __VLS_46 = __VLS_45({
        ...{ 'onClick': {} },
        variant: "ghost",
        size: "icon",
        ...{ class: "rounded-full bg-muted/50 scale-125 right-4 top-1/2 absolute backdrop-blur-xs -translate-y-1/2" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_45));
    let __VLS_49;
    const __VLS_50 = {
        /** @type {typeof __VLS_49.click} */
        onClick: (__VLS_ctx.handleNext),
    };
    /** @type {__VLS_StyleScopedClasses['rounded-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
    /** @type {__VLS_StyleScopedClasses['scale-125']} */ ;
    /** @type {__VLS_StyleScopedClasses['right-4']} */ ;
    /** @type {__VLS_StyleScopedClasses['top-1/2']} */ ;
    /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    /** @type {__VLS_StyleScopedClasses['backdrop-blur-xs']} */ ;
    /** @type {__VLS_StyleScopedClasses['-translate-y-1/2']} */ ;
    const { default: __VLS_51 } = __VLS_47.slots;
    const __VLS_52 = Icon;
    // @ts-ignore
    const __VLS_53 = __VLS_asFunctionalComponent1(__VLS_52, new __VLS_52({
        name: "i-lucide:chevron-right",
        ...{ class: "size-6" },
    }));
    const __VLS_54 = __VLS_53({
        name: "i-lucide:chevron-right",
        ...{ class: "size-6" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_53));
    /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
    // @ts-ignore
    [handleNext,];
    var __VLS_47;
    var __VLS_48;
}
const __VLS_57 = ButtonGroup || ButtonGroup;
// @ts-ignore
const __VLS_58 = __VLS_asFunctionalComponent1(__VLS_57, new __VLS_57({
    ...{ class: "scale-125 bottom-4 left-1/2 absolute backdrop-blur-xs -translate-x-1/2" },
}));
const __VLS_59 = __VLS_58({
    ...{ class: "scale-125 bottom-4 left-1/2 absolute backdrop-blur-xs -translate-x-1/2" },
}, ...__VLS_functionalComponentArgsRest(__VLS_58));
/** @type {__VLS_StyleScopedClasses['scale-125']} */ ;
/** @type {__VLS_StyleScopedClasses['bottom-4']} */ ;
/** @type {__VLS_StyleScopedClasses['left-1/2']} */ ;
/** @type {__VLS_StyleScopedClasses['absolute']} */ ;
/** @type {__VLS_StyleScopedClasses['backdrop-blur-xs']} */ ;
/** @type {__VLS_StyleScopedClasses['-translate-x-1/2']} */ ;
const { default: __VLS_62 } = __VLS_60.slots;
const __VLS_63 = Button || Button;
// @ts-ignore
const __VLS_64 = __VLS_asFunctionalComponent1(__VLS_63, new __VLS_63({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}));
const __VLS_65 = __VLS_64({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_64));
let __VLS_68;
const __VLS_69 = {
    /** @type {typeof __VLS_68.click} */
    onClick: (__VLS_ctx.handleZoomIn),
};
/** @type {__VLS_StyleScopedClasses['border-none']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
const { default: __VLS_70 } = __VLS_66.slots;
const __VLS_71 = Icon;
// @ts-ignore
const __VLS_72 = __VLS_asFunctionalComponent1(__VLS_71, new __VLS_71({
    name: "i-carbon:zoom-in",
    ...{ class: "size-5" },
}));
const __VLS_73 = __VLS_72({
    name: "i-carbon:zoom-in",
    ...{ class: "size-5" },
}, ...__VLS_functionalComponentArgsRest(__VLS_72));
/** @type {__VLS_StyleScopedClasses['size-5']} */ ;
// @ts-ignore
[handleZoomIn,];
var __VLS_66;
var __VLS_67;
const __VLS_76 = Button || Button;
// @ts-ignore
const __VLS_77 = __VLS_asFunctionalComponent1(__VLS_76, new __VLS_76({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}));
const __VLS_78 = __VLS_77({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_77));
let __VLS_81;
const __VLS_82 = {
    /** @type {typeof __VLS_81.click} */
    onClick: (__VLS_ctx.handleZoomOut),
};
/** @type {__VLS_StyleScopedClasses['border-none']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
const { default: __VLS_83 } = __VLS_79.slots;
const __VLS_84 = Icon;
// @ts-ignore
const __VLS_85 = __VLS_asFunctionalComponent1(__VLS_84, new __VLS_84({
    name: "i-carbon:zoom-out",
    ...{ class: "size-5" },
}));
const __VLS_86 = __VLS_85({
    name: "i-carbon:zoom-out",
    ...{ class: "size-5" },
}, ...__VLS_functionalComponentArgsRest(__VLS_85));
/** @type {__VLS_StyleScopedClasses['size-5']} */ ;
// @ts-ignore
[handleZoomOut,];
var __VLS_79;
var __VLS_80;
const __VLS_89 = Button || Button;
// @ts-ignore
const __VLS_90 = __VLS_asFunctionalComponent1(__VLS_89, new __VLS_89({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}));
const __VLS_91 = __VLS_90({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_90));
let __VLS_94;
const __VLS_95 = {
    /** @type {typeof __VLS_94.click} */
    onClick: (__VLS_ctx.handleOriginalSize),
};
/** @type {__VLS_StyleScopedClasses['border-none']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
const { default: __VLS_96 } = __VLS_92.slots;
const __VLS_97 = Icon;
// @ts-ignore
const __VLS_98 = __VLS_asFunctionalComponent1(__VLS_97, new __VLS_97({
    name: "i-lucide:maximize",
    ...{ class: "size-5" },
}));
const __VLS_99 = __VLS_98({
    name: "i-lucide:maximize",
    ...{ class: "size-5" },
}, ...__VLS_functionalComponentArgsRest(__VLS_98));
/** @type {__VLS_StyleScopedClasses['size-5']} */ ;
// @ts-ignore
[handleOriginalSize,];
var __VLS_92;
var __VLS_93;
const __VLS_102 = Button || Button;
// @ts-ignore
const __VLS_103 = __VLS_asFunctionalComponent1(__VLS_102, new __VLS_102({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}));
const __VLS_104 = __VLS_103({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_103));
let __VLS_107;
const __VLS_108 = {
    /** @type {typeof __VLS_107.click} */
    onClick: (__VLS_ctx.handleRotateLeft),
};
/** @type {__VLS_StyleScopedClasses['border-none']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
const { default: __VLS_109 } = __VLS_105.slots;
const __VLS_110 = Icon;
// @ts-ignore
const __VLS_111 = __VLS_asFunctionalComponent1(__VLS_110, new __VLS_110({
    name: "i-carbon:rotate",
    ...{ class: "size-5" },
}));
const __VLS_112 = __VLS_111({
    name: "i-carbon:rotate",
    ...{ class: "size-5" },
}, ...__VLS_functionalComponentArgsRest(__VLS_111));
/** @type {__VLS_StyleScopedClasses['size-5']} */ ;
// @ts-ignore
[handleRotateLeft,];
var __VLS_105;
var __VLS_106;
const __VLS_115 = Button || Button;
// @ts-ignore
const __VLS_116 = __VLS_asFunctionalComponent1(__VLS_115, new __VLS_115({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}));
const __VLS_117 = __VLS_116({
    ...{ 'onClick': {} },
    variant: "outline",
    size: "icon",
    ...{ class: "border-none bg-muted/50" },
}, ...__VLS_functionalComponentArgsRest(__VLS_116));
let __VLS_120;
const __VLS_121 = {
    /** @type {typeof __VLS_120.click} */
    onClick: (__VLS_ctx.handleRotateRight),
};
/** @type {__VLS_StyleScopedClasses['border-none']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-muted/50']} */ ;
const { default: __VLS_122 } = __VLS_118.slots;
const __VLS_123 = Icon;
// @ts-ignore
const __VLS_124 = __VLS_asFunctionalComponent1(__VLS_123, new __VLS_123({
    name: "i-carbon:rotate-180",
    ...{ class: "size-5" },
}));
const __VLS_125 = __VLS_124({
    name: "i-carbon:rotate-180",
    ...{ class: "size-5" },
}, ...__VLS_functionalComponentArgsRest(__VLS_124));
/** @type {__VLS_StyleScopedClasses['size-5']} */ ;
// @ts-ignore
[handleRotateRight,];
var __VLS_118;
var __VLS_119;
// @ts-ignore
[];
var __VLS_60;
// @ts-ignore
[];
var __VLS_10;
var __VLS_11;
// @ts-ignore
[];
var __VLS_3;
// @ts-ignore
[];
const __VLS_export = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
});
export default {};
