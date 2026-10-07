import axios from 'axios';
import { filesize } from 'filesize';
import { computed, onMounted, onUnmounted, ref, useTemplateRef } from 'vue';
import ButtonGroup from '../button/ButtonGroup.vue';
import Button from '../button/index.vue';
import Icon from '../icon/index.vue';
defineOptions({
    name: 'BuiltInFileUpload',
});
const props = withDefaults(defineProps(), {
    action: '',
    method: 'post',
    headers: () => ({}),
    data: () => ({}),
    name: 'file',
    multiple: false,
    max: 0,
    directory: false,
    disabled: false,
    description: '拖放或点击上传',
});
const emits = defineEmits();
const fileList = defineModel('modelValue', { required: true });
const containerRef = useTemplateRef('containerRef');
const fileInputRef = useTemplateRef('fileInputRef');
const isDragging = ref(false);
const isHoveringContainer = ref(false);
const isContainerFocused = ref(false);
const isMaxReached = computed(() => props.max > 0 && fileList.value.length >= props.max);
const canHandlePaste = computed(() => !props.disabled
    && !isMaxReached.value
    && !props.directory
    && (isHoveringContainer.value || isContainerFocused.value));
function handleDragOver(e) {
    e.preventDefault();
    if (props.disabled || isMaxReached.value) {
        return;
    }
    isDragging.value = true;
}
function handleDragLeave(e) {
    e.preventDefault();
    isDragging.value = false;
}
function handleDrop(e) {
    e.preventDefault();
    if (props.disabled || isMaxReached.value) {
        return;
    }
    isDragging.value = false;
    if (e.dataTransfer?.files) {
        handleFiles(e.dataTransfer.files);
    }
}
function onPaste(e) {
    if (e.defaultPrevented || !canHandlePaste.value) {
        return;
    }
    const filesFromItems = [...(e.clipboardData?.items ?? [])]
        .filter(item => item.kind === 'file')
        .map(item => item.getAsFile())
        .filter((file) => file !== null);
    const files = filesFromItems.length > 0
        ? filesFromItems
        : [...(e.clipboardData?.files ?? [])];
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
function onSelectFile(files) {
    handleFiles(files);
}
function handleFiles(files) {
    if (!files || props.disabled || isMaxReached.value) {
        return false;
    }
    const selectedFiles = [...files].filter(file => file instanceof File);
    const remain = props.max > 0 ? props.max - fileList.value.length : selectedFiles.length;
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
async function uploadFile(file, index) {
    const canUpload = await props.beforeUpload?.(file);
    if (canUpload === false) {
        return;
    }
    if (index === undefined) {
        fileList.value.push({
            name: file.name,
            size: file.size,
            status: 'uploading',
            progress: 0,
            file,
        });
    }
    const currentFileIndex = index ?? fileList.value.length - 1;
    try {
        const response = await (props.httpRequest ?? defaultHttpRequest)({
            action: props.action,
            method: props.method,
            headers: props.headers,
            data: props.data,
            name: props.name,
            file,
            onProgress: (percent) => {
                fileList.value[currentFileIndex].progress = percent;
            },
        });
        const url = await props.afterUpload?.(response);
        if (url) {
            fileList.value[currentFileIndex].url = url;
        }
        emits('onSuccess', response, file);
        fileList.value[currentFileIndex].status = 'success';
    }
    catch {
        fileList.value[currentFileIndex].status = 'error';
    }
}
function removeFile(idx) {
    fileList.value.splice(idx, 1);
}
let __VLS_modelEmit;
const __VLS_defaults = {
    action: '',
    method: 'post',
    headers: () => ({}),
    data: () => ({}),
    name: 'file',
    multiple: false,
    max: 0,
    directory: false,
    disabled: false,
    description: '拖放或点击上传',
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
__VLS_asFunctionalElement1(__VLS_intrinsics.button, __VLS_intrinsics.button)({
    ...{ onDragover: (__VLS_ctx.handleDragOver) },
    ...{ onDragleave: (__VLS_ctx.handleDragLeave) },
    ...{ onDrop: (__VLS_ctx.handleDrop) },
    ...{ onClick: (...[$event]) => {
            return (__VLS_ctx.fileInputRef?.click());
            // @ts-ignore
            [onContainerFocusIn, onContainerFocusOut, handleDragOver, handleDragLeave, handleDrop, fileInputRef,];
        } },
    type: "button",
    ...{ class: "p-4 border border-2 rounded-lg border-dashed bg-transparent flex flex-col h-40 w-full cursor-pointer transition-all items-center justify-center" },
    ...{ class: ({
            'border-primary bg-primary/5': __VLS_ctx.isDragging,
            'opacity-50 cursor-not-allowed': props.disabled || __VLS_ctx.isMaxReached,
        }) },
    disabled: (props.disabled || __VLS_ctx.isMaxReached),
});
/** @type {__VLS_StyleScopedClasses['p-4']} */ ;
/** @type {__VLS_StyleScopedClasses['border']} */ ;
/** @type {__VLS_StyleScopedClasses['border-2']} */ ;
/** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
/** @type {__VLS_StyleScopedClasses['border-dashed']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-transparent']} */ ;
/** @type {__VLS_StyleScopedClasses['flex']} */ ;
/** @type {__VLS_StyleScopedClasses['flex-col']} */ ;
/** @type {__VLS_StyleScopedClasses['h-40']} */ ;
/** @type {__VLS_StyleScopedClasses['w-full']} */ ;
/** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
/** @type {__VLS_StyleScopedClasses['transition-all']} */ ;
/** @type {__VLS_StyleScopedClasses['items-center']} */ ;
/** @type {__VLS_StyleScopedClasses['justify-center']} */ ;
/** @type {__VLS_StyleScopedClasses['border-primary']} */ ;
/** @type {__VLS_StyleScopedClasses['bg-primary/5']} */ ;
/** @type {__VLS_StyleScopedClasses['opacity-50']} */ ;
/** @type {__VLS_StyleScopedClasses['cursor-not-allowed']} */ ;
var __VLS_0 = {};
const __VLS_2 = Icon;
// @ts-ignore
const __VLS_3 = __VLS_asFunctionalComponent1(__VLS_2, new __VLS_2({
    name: "i-icon-park-outline:upload",
    ...{ class: "text-2xl text-card-foreground/50 mb-2" },
}));
const __VLS_4 = __VLS_3({
    name: "i-icon-park-outline:upload",
    ...{ class: "text-2xl text-card-foreground/50 mb-2" },
}, ...__VLS_functionalComponentArgsRest(__VLS_3));
/** @type {__VLS_StyleScopedClasses['text-2xl']} */ ;
/** @type {__VLS_StyleScopedClasses['text-card-foreground/50']} */ ;
/** @type {__VLS_StyleScopedClasses['mb-2']} */ ;
__VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
    ...{ class: "text-sm text-card-foreground/70" },
});
/** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
/** @type {__VLS_StyleScopedClasses['text-card-foreground/70']} */ ;
(props.description);
__VLS_asFunctionalElement1(__VLS_intrinsics.input)({
    ...{ onChange: (e => __VLS_ctx.onSelectFile(e.target.files)) },
    ref: "fileInputRef",
    type: "file",
    multiple: (props.directory || props.multiple),
    webkitdirectory: (props.directory || undefined),
    directory: (props.directory || undefined),
    disabled: (props.disabled),
    ...{ class: "hidden" },
});
/** @type {__VLS_StyleScopedClasses['hidden']} */ ;
if (__VLS_ctx.fileList.length > 0) {
    __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
        ...{ class: "gap-2 grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))]" },
    });
    /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
    /** @type {__VLS_StyleScopedClasses['grid']} */ ;
    /** @type {__VLS_StyleScopedClasses['grid-cols-[repeat(auto-fill,minmax(200px,1fr))]']} */ ;
    for (const [item, index] of __VLS_vFor((__VLS_ctx.fileList))) {
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ onClick: (...[$event]) => {
                    if (!(__VLS_ctx.fileList.length > 0))
                        throw 0;
                    return (__VLS_ctx.emits('onClick', item, index));
                    // @ts-ignore
                    [isDragging, isMaxReached, isMaxReached, onSelectFile, fileList, fileList, emits,];
                } },
            key: (item.name + index),
            ...{ class: "group/file-upload-item py-2 pe-2 ps-3 border rounded-lg flex gap-2 items-center relative" },
            ...{ class: ([
                    item.status === 'error' ? 'border-red-500 bg-red-500/10' : '',
                ]) },
        });
        /** @type {__VLS_StyleScopedClasses['group/file-upload-item']} */ ;
        /** @type {__VLS_StyleScopedClasses['py-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['pe-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['ps-3']} */ ;
        /** @type {__VLS_StyleScopedClasses['border']} */ ;
        /** @type {__VLS_StyleScopedClasses['rounded-lg']} */ ;
        /** @type {__VLS_StyleScopedClasses['flex']} */ ;
        /** @type {__VLS_StyleScopedClasses['gap-2']} */ ;
        /** @type {__VLS_StyleScopedClasses['items-center']} */ ;
        /** @type {__VLS_StyleScopedClasses['relative']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "flex-1 truncate" },
        });
        /** @type {__VLS_StyleScopedClasses['flex-1']} */ ;
        /** @type {__VLS_StyleScopedClasses['truncate']} */ ;
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "text-sm font-medium truncate" },
        });
        /** @type {__VLS_StyleScopedClasses['text-sm']} */ ;
        /** @type {__VLS_StyleScopedClasses['font-medium']} */ ;
        /** @type {__VLS_StyleScopedClasses['truncate']} */ ;
        (item.name);
        __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
            ...{ class: "text-xs text-card-foreground/50" },
        });
        /** @type {__VLS_StyleScopedClasses['text-xs']} */ ;
        /** @type {__VLS_StyleScopedClasses['text-card-foreground/50']} */ ;
        (__VLS_ctx.filesize(item.size, { standard: 'jedec' }));
        if (item.status === 'uploading') {
            __VLS_asFunctionalElement1(__VLS_intrinsics.div)({
                ...{ class: "bg-primary/5 pointer-events-none inset-0 absolute z-0" },
                ...{ style: ({ width: `${item.progress}%` }) },
            });
            /** @type {__VLS_StyleScopedClasses['bg-primary/5']} */ ;
            /** @type {__VLS_StyleScopedClasses['pointer-events-none']} */ ;
            /** @type {__VLS_StyleScopedClasses['inset-0']} */ ;
            /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
            /** @type {__VLS_StyleScopedClasses['z-0']} */ ;
        }
        else if (item.status === 'success') {
            __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                ...{ class: "flex-center size-10" },
            });
            /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
            /** @type {__VLS_StyleScopedClasses['size-10']} */ ;
            const __VLS_7 = Icon;
            // @ts-ignore
            const __VLS_8 = __VLS_asFunctionalComponent1(__VLS_7, new __VLS_7({
                name: "i-ix:upload-success",
                ...{ class: "text-lg text-green-600" },
            }));
            const __VLS_9 = __VLS_8({
                name: "i-ix:upload-success",
                ...{ class: "text-lg text-green-600" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_8));
            /** @type {__VLS_StyleScopedClasses['text-lg']} */ ;
            /** @type {__VLS_StyleScopedClasses['text-green-600']} */ ;
        }
        else if (item.status === 'error') {
            __VLS_asFunctionalElement1(__VLS_intrinsics.div, __VLS_intrinsics.div)({
                ...{ class: "flex-center size-10" },
            });
            /** @type {__VLS_StyleScopedClasses['flex-center']} */ ;
            /** @type {__VLS_StyleScopedClasses['size-10']} */ ;
            const __VLS_12 = Icon;
            // @ts-ignore
            const __VLS_13 = __VLS_asFunctionalComponent1(__VLS_12, new __VLS_12({
                name: "i-ix:upload-fail",
                ...{ class: "text-lg text-red-600" },
            }));
            const __VLS_14 = __VLS_13({
                name: "i-ix:upload-fail",
                ...{ class: "text-lg text-red-600" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_13));
            /** @type {__VLS_StyleScopedClasses['text-lg']} */ ;
            /** @type {__VLS_StyleScopedClasses['text-red-600']} */ ;
        }
        if (!props.disabled) {
            const __VLS_17 = ButtonGroup || ButtonGroup;
            // @ts-ignore
            const __VLS_18 = __VLS_asFunctionalComponent1(__VLS_17, new __VLS_17({
                ...{ class: "opacity-0 transition-opacity inset-e-2 top-1/2 absolute group-hover/file-upload-item:opacity-100 -translate-y-1/2" },
            }));
            const __VLS_19 = __VLS_18({
                ...{ class: "opacity-0 transition-opacity inset-e-2 top-1/2 absolute group-hover/file-upload-item:opacity-100 -translate-y-1/2" },
            }, ...__VLS_functionalComponentArgsRest(__VLS_18));
            /** @type {__VLS_StyleScopedClasses['opacity-0']} */ ;
            /** @type {__VLS_StyleScopedClasses['transition-opacity']} */ ;
            /** @type {__VLS_StyleScopedClasses['inset-e-2']} */ ;
            /** @type {__VLS_StyleScopedClasses['top-1/2']} */ ;
            /** @type {__VLS_StyleScopedClasses['absolute']} */ ;
            /** @type {__VLS_StyleScopedClasses['group-hover/file-upload-item:opacity-100']} */ ;
            /** @type {__VLS_StyleScopedClasses['-translate-y-1/2']} */ ;
            const { default: __VLS_22 } = __VLS_20.slots;
            if (item.status === 'error') {
                const __VLS_23 = Button || Button;
                // @ts-ignore
                const __VLS_24 = __VLS_asFunctionalComponent1(__VLS_23, new __VLS_23({
                    ...{ 'onClick': {} },
                    variant: "outline",
                    size: "icon",
                }));
                const __VLS_25 = __VLS_24({
                    ...{ 'onClick': {} },
                    variant: "outline",
                    size: "icon",
                }, ...__VLS_functionalComponentArgsRest(__VLS_24));
                let __VLS_28;
                const __VLS_29 = {
                    /** @type {typeof __VLS_28.click} */
                    onClick: (...[$event]) => {
                        if (!(__VLS_ctx.fileList.length > 0))
                            throw 0;
                        if (!(!props.disabled))
                            throw 0;
                        if (!(item.status === 'error'))
                            throw 0;
                        return (__VLS_ctx.uploadFile(item.file, index));
                        // @ts-ignore
                        [filesize, uploadFile,];
                    },
                };
                const { default: __VLS_30 } = __VLS_26.slots;
                const __VLS_31 = Icon;
                // @ts-ignore
                const __VLS_32 = __VLS_asFunctionalComponent1(__VLS_31, new __VLS_31({
                    name: "i-icon-park-outline:upload",
                    ...{ class: "text-lg cursor-pointer" },
                }));
                const __VLS_33 = __VLS_32({
                    name: "i-icon-park-outline:upload",
                    ...{ class: "text-lg cursor-pointer" },
                }, ...__VLS_functionalComponentArgsRest(__VLS_32));
                /** @type {__VLS_StyleScopedClasses['text-lg']} */ ;
                /** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
                // @ts-ignore
                [];
                var __VLS_26;
                var __VLS_27;
            }
            if (item.status !== 'uploading') {
                const __VLS_36 = Button || Button;
                // @ts-ignore
                const __VLS_37 = __VLS_asFunctionalComponent1(__VLS_36, new __VLS_36({
                    ...{ 'onClick': {} },
                    variant: "outline",
                    size: "icon",
                }));
                const __VLS_38 = __VLS_37({
                    ...{ 'onClick': {} },
                    variant: "outline",
                    size: "icon",
                }, ...__VLS_functionalComponentArgsRest(__VLS_37));
                let __VLS_41;
                const __VLS_42 = {
                    /** @type {typeof __VLS_41.click} */
                    onClick: (...[$event]) => {
                        if (!(__VLS_ctx.fileList.length > 0))
                            throw 0;
                        if (!(!props.disabled))
                            throw 0;
                        if (!(item.status !== 'uploading'))
                            throw 0;
                        return (__VLS_ctx.removeFile(index));
                        // @ts-ignore
                        [removeFile,];
                    },
                };
                const { default: __VLS_43 } = __VLS_39.slots;
                const __VLS_44 = Icon;
                // @ts-ignore
                const __VLS_45 = __VLS_asFunctionalComponent1(__VLS_44, new __VLS_44({
                    name: "i-icon-park-outline:delete",
                    ...{ class: "text-lg text-red-500 cursor-pointer" },
                }));
                const __VLS_46 = __VLS_45({
                    name: "i-icon-park-outline:delete",
                    ...{ class: "text-lg text-red-500 cursor-pointer" },
                }, ...__VLS_functionalComponentArgsRest(__VLS_45));
                /** @type {__VLS_StyleScopedClasses['text-lg']} */ ;
                /** @type {__VLS_StyleScopedClasses['text-red-500']} */ ;
                /** @type {__VLS_StyleScopedClasses['cursor-pointer']} */ ;
                // @ts-ignore
                [];
                var __VLS_39;
                var __VLS_40;
            }
            // @ts-ignore
            [];
            var __VLS_20;
        }
        // @ts-ignore
        [];
    }
}
// @ts-ignore
var __VLS_1 = __VLS_0;
// @ts-ignore
[];
const __VLS_base = (await import('vue')).defineComponent({
    __typeEmits: {},
    __typeProps: {},
    props: {},
});
const __VLS_export = {};
export default {};
