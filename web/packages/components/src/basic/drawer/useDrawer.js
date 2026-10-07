import { createVNode, getCurrentInstance, h, isVNode, onUnmounted, ref, render, shallowRef, watchEffect } from 'vue';
import Drawer from './index.vue';
export function useDrawer() {
    function create(initialOptions) {
        const container = document.createElement('div');
        const visible = ref(false);
        const options = shallowRef({ ...initialOptions });
        const instance = getCurrentInstance();
        let vnode = null;
        const updateVNode = () => {
            vnode = createVNode(Drawer, Object.assign({
                'id': instance && instance.uid ? `FaDrawer-${instance.uid}` : undefined,
                'modelValue': visible.value,
                'onUpdate:modelValue': (val) => {
                    visible.value = val;
                },
                ...options.value,
            }), {
                default: () => {
                    if (typeof options.value.content === 'string') {
                        return options.value.content;
                    }
                    else if (isVNode(options.value.content)) {
                        return options.value.content;
                    }
                    else if (options.value.content) {
                        return h(options.value.content);
                    }
                    return null;
                },
            });
            // 继承主应用的上下文
            if (instance && instance.appContext) {
                vnode.appContext = instance.appContext;
            }
            render(vnode, container);
        };
        // 监听 visible 和 options 变化，自动重新渲染
        watchEffect(() => {
            updateVNode();
        });
        // 挂载到当前实例
        instance?.proxy?.$el?.appendChild(container);
        // 监听组件卸载，自动清理
        if (instance) {
            onUnmounted(() => {
                if (vnode) {
                    render(null, container);
                    vnode = null;
                }
                if (container.parentNode) {
                    container.parentNode.removeChild(container);
                }
            });
        }
        const open = () => {
            visible.value = true;
        };
        const close = () => {
            visible.value = false;
        };
        const update = (newOptions) => {
            options.value = { ...options.value, ...newOptions };
        };
        return {
            open,
            close,
            update,
        };
    }
    return {
        create,
    };
}
