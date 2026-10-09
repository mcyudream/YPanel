/**
 * 内置适配器样式（菜单/对话框/toast）。
 */
export const builtinAdapterCss = `
.yw-menu-host { position: fixed; inset: 0; z-index: 9999; pointer-events: none; }
.yw-menu {
  position: absolute; min-width: 180px; padding: 5px;
  background: oklch(var(--yw-popover, 1 0 0));
  border: 1px solid oklch(var(--yw-border, 0.9 0.01 255));
  border-radius: var(--yw-radius-menu, 8px);
  box-shadow: var(--yw-shadow-menu, 0 10px 34px rgba(0,0,0,.28));
  pointer-events: auto;
  animation: yw-menu-in 0.12s ease;
}
@keyframes yw-menu-in { from { opacity: 0; transform: translateY(-4px) } to { opacity: 1; transform: none } }
.yw-menu-item {
  display: flex; align-items: center; gap: 8px;
  height: 22px; padding: 0 10px; font-size: 13px; cursor: default; user-select: none;
  color: oklch(var(--yw-popover-foreground, 0.21 0.02 255));
  border-radius: var(--yw-radius-menu-item, 5px);
}
.yw-menu-item:hover { background: oklch(var(--yw-primary) / 90%); color: #fff; }
.yw-menu-item--danger { color: oklch(var(--yw-destructive, 0.6 0.22 27)); }
.yw-menu-item--danger:hover { background: oklch(var(--yw-destructive, 0.6 0.22 27)); color: #fff; }
.yw-menu-item--disabled { opacity: 0.4; pointer-events: none; }
.yw-menu-item-icon { width: 16px; text-align: center; }
.yw-menu-caret { margin-left: auto; opacity: 0.6; }

.yw-overlay-host { position: fixed; inset: 0; z-index: 9998; }
.yw-overlay {
  position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
  background: rgb(0 0 0 / 40%); backdrop-filter: blur(2px);
}
.yw-dialog {
  min-width: 320px; max-width: 480px; padding: 20px;
  background: oklch(var(--yw-popover, 1 0 0));
  border: 1px solid oklch(var(--yw-border, 0.9 0.01 255));
  border-radius: var(--yw-radius-window, 16px);
  box-shadow: var(--yw-shadow-window, 0 24px 72px rgba(0,0,0,.35));
}
.yw-dialog-title { font-size: 15px; font-weight: 600; color: oklch(var(--yw-popover-foreground, 0.21 0.02 255)); }
.yw-dialog-content { margin-top: 10px; font-size: 13px; color: oklch(var(--yw-muted-foreground, 0.55 0.02 255)); }
.yw-dialog-actions { display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px; }
.yw-btn {
  padding: 6px 16px; font-size: 13px; cursor: pointer;
  border: 1px solid oklch(var(--yw-border, 0.9 0.01 255)); border-radius: var(--yw-radius-button, 8px);
  background: oklch(var(--yw-secondary, 0.94 0.01 255));
  color: oklch(var(--yw-secondary-foreground, 0.21 0.02 255));
}
.yw-btn--primary { background: oklch(var(--yw-primary, 0.55 0.19 262)); color: #fff; border-color: transparent; }
.yw-btn--danger { background: oklch(var(--yw-destructive, 0.6 0.22 27)); color: #fff; border-color: transparent; }

[data-yw-toasts] {
  position: fixed; top: 16px; right: 16px; z-index: 10000;
  display: flex; flex-direction: column; gap: 8px; pointer-events: none;
}
.yw-toast {
  padding: 10px 16px; font-size: 13px; color: #fff;
  background: rgb(30 41 59 / 92%); border-radius: 10px;
  box-shadow: 0 8px 24px -6px rgb(0 0 0 / 40%);
  opacity: 0; transform: translateX(12px); transition: all 0.22s ease;
}
.yw-toast.is-in { opacity: 1; transform: none; }
.yw-toast--success { background: rgb(22 163 74 / 92%); }
.yw-toast--error { background: rgb(220 38 38 / 92%); }
.yw-toast--warning { background: rgb(217 119 6 / 92%); }
`
