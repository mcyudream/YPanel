import axios from 'axios';
import { computed, onMounted, onUnmounted, ref, shallowRef, useTemplateRef, watch } from 'vue';
import { cn } from '#utils';
import Icon from '../icon/index.vue';
import { useImagePreview } from '../image-preview';
import Progress from '../progress/index.vue';
defineOptions({
    name: 'BuiltInImageUpload',
});
const props = withDefaults(defineProps(), {
    action: '',
    method: 'post',
    headers: () => ({}),
    data: () => ({}),
    name: 'file',
    multiple: false,
    max: 1,
    width: 100,
    height: 100,
    directory: false,
    disabled: false,
});
const emits = defineEmits();
const images = defineModel('modelValue', { required: true });
const activeUploadCount = ref(0);
const uploadProgress = ref(0);
const imageItemSeed = shallowRef(0);
const imageItems = ref([]);
const containerRef = useTemplateRef('containerRef');
const fileInputRef = useTemplateRef('fileInputRef');
const isHoveringContainer = ref(false);
const isContainerFocused = ref(false);
const isUploading = computed(() => activeUploadCount.value > 0);
const canHandlePaste = computed(() => !props.disabled
    && !isUploading.value
    && !props.directory
    && (isHoveringContainer.value || isContainerFocused.value));
watch(images, (currentImages) => {
    const reusableItems = new Map();
    imageItems.value.forEach((item) => {
        const itemsForSrc = reusableItems.get(item.src);
        if (itemsForSrc) {
            itemsForSrc.push(item);
        }
        else {
            reusableItems.set(item.src, [item]);
        }
    });
    imageItems.value = currentImages.map((src) => {
        const reusableItem = reusableItems.get(src)?.shift();
        if (reusableItem) {
            return reusableItem;
        }
        imageItemSeed.value += 1;
        return {
            key: `fa-image-upload-item-${imageItemSeed.value}`,
            src,
        };
    });
}, {
    deep: true,
    immediate: true,
});
function getHeadersObject(headers) {
    if (!(headers instanceof Headers)) {
        return { ...headers };
    }
    const headersObj = {};
    headers.forEach((value, key) => {
        headersObj[key] = value;
    });
    return headersObj;
}
async function defaultHttpRequest(options) {
    const formData = new FormData();
    Object.entries(options.data).forEach(([key, value]) => {
        formData.append(key, value);
    });
    formData.append(options.name, options.file);
    const response = await axios({
        url: options.action,
        method: options.method,
        headers: getHeadersObject(options.headers),
        data: formData,
        onUploadProgress: (progressEvent) => {
            if (progressEvent.total) {
                options.onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total));
            }
        },
    });
    return response.data;
}
async function uploadFile(file) {
    const canUpload = await props.beforeUpload?.(file);
    if (canUpload === false) {
        return;
    }
    activeUploadCount.value += 1;
    uploadProgress.value = 0;
    try {
        const response = await (props.httpRequest ?? defaultHttpRequest)({
            action: props.action,
            method: props.method,
            headers: props.headers,
            data: props.data,
            name: props.name,
            file,
            onProgress: (percent) => {
                uploadProgress.value = percent;
            },
        });
        const url = await props.afterUpload?.(response);
        if (url) {
            images.value.push(url);
        }
        emits('onSuccess', response, file);
    }
    finally {
        activeUploadCount.value -= 1;
        if (!isUploading.value) {
            uploadProgress.value = 0;
        }
    }
}
function onSelectFile(e) {
    handleFiles(e.target.files);
}
function onPaste(e) {
    if (e.defaultPrevented || !canHandlePaste.value) {
        return;
    }
    const clipboardItems = [...(e.clipboardData?.items ?? [])];
    const filesFromItems = clipboardItems
        .filter(item => item.kind === 'file' && item.type.startsWith('image/'))
        .map(item => item.getAsFile())
        .filter((file) => file !== null);
    const files = filesFromItems.length > 0
        ? filesFromItems
        : [...(e.clipboardData?.files ?? [])].filter(file => file.type.startsWith('image/'));
    if (files.length > 0 && handleFiles(files)) {
        e.preventDefault();
    }
}
function onContainerFocusIn() {
    isContainerFocused.value = true;
}
function onContainerFocusOut(e) {
    const nextFocusedElement = e.relatedTarget;
    if (nextFocusedElement instanceof Node && containerRef.value?.contains(nextFocusedElement)) {
        return;
    }
    isContainerFocused.value = false;
}
onMounted(() => {
    window.addEventListener('paste', onPaste);
});
onUnmounted(() => {
    window.removeEventListener('paste', onPaste);
});
function handleFiles(files) {
    if (!files || props.disabled || isUploading.value) {
        return false;
    }
    const selectedFiles = [...files].filter(file => file instanceof File);
    const remain = props.max === 0 ? selectedFiles.length : props.max - images.value.length;
    if (remain <= 0) {
        return false;
    }
    if (fileInputRef.value) {
        fileInputRef.value.value = '';
    }
    const filesToAdd = selectedFiles.slice(0, remain);
    if (filesToAdd.length === 0) {
        return false;
    }
    filesToAdd.forEach((file) => {
        void uploadFile(file);
    });
    return true;
}
function onRemove(idx) {
    images.value.splice(idx, 1);
}
function onPreview(index) {
    useImagePreview().open(images.value, index);
}
function onMove(index, direction) {
    if (direction === 'forward' && index !== 0) {
        images.value[index] = images.value.splice(index - 1, 1, images.value[index])[0];
    }
    if (direction === 'backward' && index !== images.value.length - 1) {
        images.value[index] = images.value.splice(index + 1, 1, images.value[index])[0];
    }
}
function onBeforeLeave(el) {
    if (!(el instanceof HTMLElement) || !(el.parentElement instanceof HTMLElement)) {
        return;
    }
    const elementRect = el.getBoundingClientRect();
    const listRect = el.parentElement.getBoundingClientRect();
    el.style.left = `${elementRect.left - listRect.left}px`;
    el.style.top = `${elementRect.top - listRect.top}px`;
    el.style.width = `${elementRect.width}px`;
    el.style.height = `${elementRect.height}px`;
}
let __VLS_modelEmit;
const __VLS_defaults = {
    action: '',
    method: 'post',
    headers: () => ({}),
    data: () => ({}),
    name: 'file',
    multiple: false,
    max: 1,
    width: 100,
    height: 100,
    directory: false,
    disabled: false,
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
/** @type {__VLS_StyleScopedClasses['fa-image-upload-list-leave-active']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ onMouseenter: (...[$event]) => {
            return (__VLS_ctx.isHoveringContainer = true);
            // @ts-ignore
            [isHoveringContainer,];
        } },
    ...{ onMouseleave: (...[$event]) => {
            return (__VLS_ctx.isHoveringContainer = false);
            // @ts-ignore
            [isHoveringContainer,];
        } },
    ...{ onFocusin: (__VLS_ctx.onContainerFocusIn) },
    ...{ onFocusout: (__VLS_ctx.onContainerFocusOut) },
    ref: "containerRef",
    ...{ class: "space-y-2" },
    tabindex: (props.disabled ? undefined : 0),
});
/** @type {__VLS_StyleScopedClasses['space-y-2']} */ ;
let __VLS_0;
/** @ts-ignore @type { | typeof __VLS_components.TransitionGroup | typeof __VLS_components.TransitionGroup} */
TransitionGroup;
// @ts-ignore
const __VLS_1 = __VLS_asFunctionalComponent1(__VLS_0, new __VLS_0({
    ...{ 'onBeforeLeave': {} },
    tag: "div",
    name: "fa-image-upload-list",
    ...{ class: "fa-image-upload-list flex flex-wrap gap-2" },
}));
const __VLS_2 = __VLS_1({
    ...{ 'onBeforeLeave': {} },
    tag: "div",
    name: "fa-image-upload-list",
    ...{ class: "fa-image-upload-list flex flex-wrap gap-2" },
}, ...__VLS_functionalComponentArgsRest(__VLS_1));
let __VLS_5;
const __VLS_6 = {
    /** @type {typeof __VLS_5.beforeLeave} */
    onBeforeLeave: (__VLS_ctx.onBeforeLeave),
};
/** @type {__VLS_StyleScopedClasses['fa-image-upload-list']} */ ;
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-wrap']} */ ;
/** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
const { default: __VLS_7 } = __VLS_3.slots;
for (const [item, index] of __VLS_vFor((__VLS_ctx.imageItems))) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        key: (item.key),
        ...{ class: "fa-image-upload-item group/image-upload border rounded-lg flex items-center justify-center relative overflow-hidden" },
        ...{ style: ({
                width: `${props.width}px`,
                height: `${props.height}px`,
            }) },
    });
    /** @type {__VLS_StyleScopedClasses['fa-image-upload-item']} */ ;
    /** @type {__VLS_StyleScopedClasses['group/image-upload']} */ ;
    /** @type {__VLS_StyleScopedClasses['border']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
    /** @type {__VLS_StyleScopedClasses['flex']} */ ;
    /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['relative']} */ ;
    /** @type {__VLS_StyleScopedClasses['overflow-hidden']} */ ;
    __VLS_asFunctionalElement1(__VLS_intrinsics.img)({
        src: (item.src),
        ...{ class: "h-full w-full object-contain" },
    });
    /** @type {__VLS_StyleScopedClasses['h-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['w-full']} */ ;
    /** @type {__VLS_StyleScopedClasses['object-contain']} */ ;
    if (!props.disabled) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "text-white p-2 rounded-lg bg-black/50 opacity-0 grid grid-cols-2 transition-opacity inset-0 place-items-center absolute group-hover/image-upload:opacity-100" },
        });
        /** @type {__VLS_StyleScopedClasses['text-white']} */ ;
        /** @type {__VLS_StyleScopedClasses['p-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
        /** @type {__VLS_StyleScopedClasses['bg-black/50']} */ ;
        /** @type {__VLS_StyleScopedClasses['opacity-0']} */ ;
        /** @type {__VLS_StyleScopedClasses['grid']} */ ;
        /** @type {__VLS_StyleScopedClasses['grid-cols-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['transition-opacity']} */ ;
        /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
        /** @type {__VLS_StyleScopedClasses['place-items-center']} */ ;
        /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
        /** @type {__VLS_StyleScopedClasses['group-hover/image-upload:opacity-100']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ onClick: (...[$event]) => {
                    if (!(!props.disabled))
                        throw 0;
                    return (__VLS_ctx.onPreview(index));
                    // @ts-ignore
                    [onContainerFocusIn, onContainerFocusOut, onBeforeLeave, imageItems, onPreview,];
                } },
            ...{ class: "opacity-60 flex-center cursor-pointer transition-all hover:(opacity-100 scale-110)" },
        });
        /** @type {__VLS_StyleScopedClasses['opacity-60']} */ ;
        /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
        /** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
        /** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
        /** @type {__VLS_StyleScopedClasses['hover:(opacity-100']} */ ;
        /** @type {__VLS_StyleScopedClasses['scale-110)']} */ ;
        const __VLS_8 = Icon;
        // @ts-ignore
        const __VLS_9 = __VLS_asFunctionalComponent1(__VLS_8, new __VLS_8({
            name: "i-icon-park-outline:preview-open",
            ...{ class: "size-6" },
        }));
        const __VLS_10 = __VLS_9({
            name: "i-icon-park-outline:preview-open",
            ...{ class: "size-6" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_9));
        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ onClick: (...[$event]) => {
                    if (!(!props.disabled))
                        throw 0;
                    return (__VLS_ctx.onRemove(index));
                    // @ts-ignore
                    [onRemove,];
                } },
            ...{ class: "opacity-60 flex-center cursor-pointer transition-all hover:(opacity-100 scale-110)" },
        });
        /** @type {__VLS_StyleScopedClasses['opacity-60']} */ ;
        /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
        /** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
        /** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
        /** @type {__VLS_StyleScopedClasses['hover:(opacity-100']} */ ;
        /** @type {__VLS_StyleScopedClasses['scale-110)']} */ ;
        const __VLS_13 = Icon;
        // @ts-ignore
        const __VLS_14 = __VLS_asFunctionalComponent1(__VLS_13, new __VLS_13({
            name: "i-icon-park-outline:delete",
            ...{ class: "size-6" },
        }));
        const __VLS_15 = __VLS_14({
            name: "i-icon-park-outline:delete",
            ...{ class: "size-6" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_14));
        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ onClick: (...[$event]) => {
                    if (!(!props.disabled))
                        throw 0;
                    return (__VLS_ctx.onMove(index, 'forward'));
                    // @ts-ignore
                    [onMove,];
                } },
            ...{ class: (__VLS_ctx.cn('flex-center cursor-pointer opacity-60 transition-all', {
                    'hover:(scale-110 opacity-100)': index !== 0,
                    'cursor-not-allowed': index === 0,
                })) },
        });
        __VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (__VLS_ctx.images.length > 1), }, null, null);
        const __VLS_18 = Icon;
        // @ts-ignore
        const __VLS_19 = __VLS_asFunctionalComponent1(__VLS_18, new __VLS_18({
            name: "i-icon-park-outline:arrow-circle-left",
            ...{ class: "size-6" },
        }));
        const __VLS_20 = __VLS_19({
            name: "i-icon-park-outline:arrow-circle-left",
            ...{ class: "size-6" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_19));
        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ onClick: (...[$event]) => {
                    if (!(!props.disabled))
                        throw 0;
                    return (__VLS_ctx.onMove(index, 'backward'));
                    // @ts-ignore
                    [onMove, cn, images,];
                } },
            ...{ class: (__VLS_ctx.cn('flex-center cursor-pointer opacity-60 transition-all', {
                    'hover:(scale-110 opacity-100)': index !== __VLS_ctx.images.length - 1,
                    'cursor-not-allowed': index === __VLS_ctx.images.length - 1,
                })) },
        });
        __VLS_asFunctionalDirective(__VLS_directives.vShow, {})(null, { ...__VLS_directiveBindingRestFields, value: (__VLS_ctx.images.length > 1), }, null, null);
        const __VLS_23 = Icon;
        // @ts-ignore
        const __VLS_24 = __VLS_asFunctionalComponent1(__VLS_23, new __VLS_23({
            name: "i-icon-park-outline:arrow-circle-right",
            ...{ class: "size-6" },
        }));
        const __VLS_25 = __VLS_24({
            name: "i-icon-park-outline:arrow-circle-right",
            ...{ class: "size-6" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_24));
        /** @type {__VLS_StyleScopedClasses['size-6']} */ ;
    }
    // @ts-ignore
    [cn, images, images, images,];
}
if (__VLS_ctx.images.length < props.max || props.max === 0) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
        ...{ onClick: (...[$event]) => {
                if (!(__VLS_ctx.images.length < props.max || props.max === 0))
                    throw 0;
                return (__VLS_ctx.fileInputRef?.click());
                // @ts-ignore
                [images, fileInputRef,];
            } },
        type: "button",
        ...{ class: "bg-transparent flex-center relative overflow-hidden" },
        ...{ class: ({
                'cursor-not-allowed': props.disabled || __VLS_ctx.isUploading,
            }) },
        ...{ style: ({
                width: `${props.width}px`,
                height: `${props.height}px`,
            }) },
        disabled: (props.disabled || __VLS_ctx.isUploading),
    });
    /** @type {__VLS_StyleScopedClasses['bg-transparent']} */ ;
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['relative']} */ ;
    /** @type {__VLS_StyleScopedClasses['overflow-hidden']} */ ;
    /** @type {__VLS_StyleScopedClasses['cursor-not-allowed']} */ ;
    var __VLS_28 = {};
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "text-primary p-0 border border-2 rounded-lg border-dashed flex-center size-full" },
    });
    /** @type {__VLS_StyleScopedClasses['text-primary']} */ ;
    /** @type {__VLS_StyleScopedClasses['p-0']} */ ;
    /** @type {__VLS_StyleScopedClasses['border']} */ ;
    /** @type {__VLS_StyleScopedClasses['border-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
    /** @type {__VLS_StyleScopedClasses['border-dashed']} */ ;
    /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
    /** @type {__VLS_StyleScopedClasses['size-full']} */ ;
    const __VLS_30 = Icon;
    // @ts-ignore
    const __VLS_31 = __VLS_asFunctionalComponent1(__VLS_30, new __VLS_30({
        name: "i-icon-park-outline:upload",
        ...{ class: "text-2xl text-card-foreground/50" },
    }));
    const __VLS_32 = __VLS_31({
        name: "i-icon-park-outline:upload",
        ...{ class: "text-2xl text-card-foreground/50" },
    }, ...__VLS_functionalComponentArgsRest(__VLS_31));
    /** @type {__VLS_StyleScopedClasses['text-2xl']} */ ;
    /** @type {__VLS_StyleScopedClasses['text-card-foreground/50']} */ ;
    if (__VLS_ctx.isUploading && __VLS_ctx.uploadProgress > 0 && __VLS_ctx.uploadProgress < 100) {
        const __VLS_35 = Progress;
        // @ts-ignore
        const __VLS_36 = __VLS_asFunctionalComponent1(__VLS_35, new __VLS_35({
            modelValue: (__VLS_ctx.uploadProgress),
            ...{ class: "h-1 w-auto inset-x-1 inset-b-1 absolute" },
        }));
        const __VLS_37 = __VLS_36({
            modelValue: (__VLS_ctx.uploadProgress),
            ...{ class: "h-1 w-auto inset-x-1 inset-b-1 absolute" },
        }, ...__VLS_functionalComponentArgsRest(__VLS_36));
        /** @type {__VLS_StyleScopedClasses['h-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['w-auto']} */ ;
        /** @type {__VLS_StyleScopedClasses['inset-x-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['inset-b-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
    }
    __VLS_asFunctionalElement1(__VLS_intrinsics.input)({
        ...{ onChange: (__VLS_ctx.onSelectFile) },
        ref: "fileInputRef",
        type: "file",
        accept: "image/*",
        multiple: (props.directory || props.multiple),
        webkitdirectory: (props.directory || undefined),
        directory: (props.directory || undefined),
        ...{ class: "hidden" },
    });
    /** @type {__VLS_StyleScopedClasses['hidden']} */ ;
}
// @ts-ignore
[isUploading, isUploading, isUploading, uploadProgress, uploadProgress, uploadProgress, onSelectFile,];
var __VLS_3;
var __VLS_4;
// @ts-ignore
var __VLS_29 = __VLS_28;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
