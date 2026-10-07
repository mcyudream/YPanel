import hotkeys from 'hotkeys-js';
import { onUnmounted, toValue, watch } from 'vue';
import { BUILTIN_HOTKEY_ID, builtinGlobalHotkeyBindings, builtinMenuSearchHotkeyBindings, } from './registry';
export function createHotkeys(options) {
    const HOTKEY_ID = {
        ...BUILTIN_HOTKEY_ID,
        ...options.extendIds,
    };
    const hotkeyIds = Object.values(HOTKEY_ID);
    const extendGlobalHotkeyBindings = options.extendGlobalHotkeyBindings ?? [];
    const extendScopedHotkeyBindings = options.extendScopedHotkeyBindings ?? [];
    const globalHotkeyBindings = [
        ...builtinGlobalHotkeyBindings,
        ...extendGlobalHotkeyBindings,
    ];
    const menuSearchHotkeyBindings = [
        ...builtinMenuSearchHotkeyBindings,
    ];
    const hotkeyBindings = [
        ...globalHotkeyBindings,
        ...menuSearchHotkeyBindings,
        ...extendScopedHotkeyBindings,
    ];
    function useHotkey(id, handler, active = true) {
        const binding = hotkeyBindings.find(item => item.id === id);
        let currentHandler;
        function bind() {
            if (currentHandler || !binding) {
                return;
            }
            const currentBinding = binding;
            currentHandler = (event, hotkeyHandler) => {
                const hotkeyContext = options.getContext();
                if (currentBinding.enabled && !currentBinding.enabled(hotkeyContext)) {
                    return;
                }
                if (currentBinding.preventDefault !== false) {
                    event.preventDefault();
                }
                handler({
                    event,
                    hotkey: hotkeyHandler.key,
                });
            };
            hotkeys(currentBinding.keys.join(','), currentHandler);
        }
        function unbind() {
            if (!currentHandler || !binding) {
                return;
            }
            hotkeys.unbind(binding.keys.join(','), currentHandler);
            currentHandler = undefined;
        }
        watch(() => toValue(active), (val) => {
            if (val) {
                bind();
            }
            else {
                unbind();
            }
        }, {
            immediate: true,
        });
        onUnmounted(() => {
            unbind();
        });
    }
    function useHotkeyBindings(handlers, active = true) {
        hotkeyIds.forEach((id) => {
            const handler = handlers[id];
            if (handler) {
                useHotkey(id, handler, active);
            }
        });
    }
    if (import.meta.env.DEV) {
        const registeredIds = hotkeyBindings.map(item => item.id);
        const missingIds = hotkeyIds.filter(id => !registeredIds.includes(id));
        const duplicateIds = registeredIds.filter((id, index) => registeredIds.indexOf(id) !== index);
        if (missingIds.length || duplicateIds.length) {
            console.warn('[hotkeys] registry consistency check failed', {
                missingIds,
                duplicateIds: [...new Set(duplicateIds)],
            });
        }
    }
    return {
        HOTKEY_ID,
        hotkeyIds,
        globalHotkeyBindings,
        menuSearchHotkeyBindings,
        hotkeyBindings,
        useHotkey,
        useHotkeyBindings,
    };
}
