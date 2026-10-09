import type { ThemeTokens } from '../types/theme'

/**
 * YudreamWebOS 亮/暗主题 token —— macOS Tahoe「Liquid Glass」设计体系。
 *
 * 两类变量：
 * 1. 语义色（oklch 通道值，消费端拼 oklch(var(--yw-x))）：background/foreground/card/... 兼容 shadcn 命名
 * 2. 液态玻璃与尺寸（完整 CSS 值，直接使用）：--yw-glass-*、--yw-shadow-*、--yw-radius-*、--yw-dock-* 等
 */

export const lightTheme: ThemeTokens = {
  /* ── 语义色（oklch 通道 "L C H"）── */
  '--yw-background': '0.97 0.004 250',
  '--yw-foreground': '0.18 0.02 260',
  '--yw-card': '0.925 0.006 250',
  '--yw-card-foreground': '0.18 0.02 260',
  '--yw-popover': '0.985 0.002 250',
  '--yw-popover-foreground': '0.18 0.02 260',
  '--yw-primary': '0.603 0.218 257.4',
  '--yw-primary-foreground': '0.985 0.002 240',
  '--yw-secondary': '0.93 0.008 255',
  '--yw-secondary-foreground': '0.18 0.02 260',
  '--yw-muted': '0.94 0.006 255',
  '--yw-muted-foreground': '0.5 0.02 260',
  '--yw-accent': '0.93 0.015 255',
  '--yw-accent-foreground': '0.18 0.02 260',
  '--yw-destructive': '0.62 0.21 25',
  '--yw-destructive-foreground': '0.985 0.002 240',
  '--yw-border': '0.88 0.01 255',
  '--yw-input': '0.88 0.01 255',
  '--yw-ring': '0.603 0.218 257.4',
  '--yw-glass': '0.98 0.003 250',
  '--yw-glass-foreground': '0.18 0.02 260',
  '--yw-desktop-icon-label': '0.99 0.002 240',
  /* macOS 标签层级（对照参考站 --label-*） */
  '--yw-label-2': 'rgba(0, 0, 0, 0.5)',
  '--yw-label-3': 'rgba(0, 0, 0, 0.26)',
  '--yw-label-4': 'rgba(0, 0, 0, 0.1)',
  '--yw-separator': 'rgba(0, 0, 0, 0.1)',
  '--yw-control-bg': '#FFFFFF',
  '--yw-window-bg': '#ECECEC',
  '--yw-accent-selection': 'color-mix(in srgb, oklch(var(--yw-primary)) 28%, transparent)',

  /* ── Liquid Glass：模糊与饱和 ── */
  '--yw-blur-strong': '30px',
  '--yw-blur-medium': '20px',
  '--yw-blur-light': '12px',
  '--yw-saturate': '180%',

  /* ── 液态玻璃：发丝描边 / 顶部高光 / 内发光 ── */
  '--yw-hairline': 'rgba(255, 255, 255, 0.45)',
  '--yw-hairline-weak': 'rgba(255, 255, 255, 0.22)',
  '--yw-specular-top': 'inset 1px 1px 0 rgba(255, 255, 255, 0.55)',
  '--yw-inner-glow': 'inset 0 0 12px rgba(255, 255, 255, 0.18)',

  /* ── 阴影体系（环境影 + 接触影 + 0.5px 描边）── */
  '--yw-shadow-window': '0 24px 72px rgba(0, 0, 0, 0.28), 0 2px 12px rgba(0, 0, 0, 0.12), 0 0 0 0.5px rgba(0, 0, 0, 0.2)',
  '--yw-shadow-window-inactive': '0 10px 34px rgba(0, 0, 0, 0.16), 0 0 0 0.5px rgba(0, 0, 0, 0.12)',
  '--yw-shadow-menu': '0 10px 34px rgba(0, 0, 0, 0.18), 0 0 0 0.5px rgba(0, 0, 0, 0.08)',
  '--yw-shadow-dock': 'inset 1px 1px 0 rgba(255, 255, 255, 0.55), inset 0 0 0 0.5px rgba(255, 255, 255, 0.28), 0 12px 32px rgba(0, 0, 0, 0.22)',
  '--yw-shadow-widget': '0 10px 34px rgba(0, 0, 0, 0.14), 0 0 0 0.5px rgba(255, 255, 255, 0.35)',
  '--yw-shadow-tooltip': '0 6px 20px rgba(0, 0, 0, 0.22), 0 0 0 0.5px rgba(0, 0, 0, 0.08)',

  /* ── 玻璃面板底色 ── */
  '--yw-glass-dock-bg': 'rgba(255, 255, 255, 0.42)',
  '--yw-glass-panel-bg': 'rgba(250, 250, 252, 0.72)',
  '--yw-glass-menubar-solid': 'rgba(246, 246, 250, 0.55)',
  '--yw-glass-widget-bg': 'rgba(252, 252, 254, 0.68)',

  /* ── 圆角体系 ── */
  '--yw-radius-window': '16px',
  '--yw-radius-sidebar': '12px',
  '--yw-radius-dock': '24px',
  '--yw-radius-dock-icon': '22.37%',
  '--yw-radius-widget': '14px',
  '--yw-radius-menu': '8px',
  '--yw-radius-menu-item': '5px',
  '--yw-radius-spotlight': '20px',
  '--yw-radius-button': '8px',
  '--yw-radius-capsule': '999px',
  '--yw-radius-lg': '16px',
  '--yw-radius-md': '10px',
  '--yw-radius-sm': '6px',

  /* ── 尺寸 ── */
  '--yw-menubar-h': '24px',
  '--yw-titlebar-h': '44px',
  '--yw-tl-size': '12px',
  '--yw-tl-gap': '8px',
  '--yw-tl-inset': '13px',
  '--yw-dock-icon-size': '48px',
  '--yw-dock-pad': '7px',
  '--yw-dock-gap': '4px',
  '--yw-dock-margin': '10px',
  '--yw-deskicon-cell-w': '84px',
  '--yw-deskicon-cell-h': '92px',
  '--yw-deskicon-size': '48px',
  '--yw-resize-hit': '6px',

  /* ── 交通灯 ── */
  '--yw-tl-close': '#FF5F57',
  '--yw-tl-min': '#FEBC2E',
  '--yw-tl-zoom': '#28C840',
  '--yw-tl-inactive': '#D9D9DC',

  /* ── 动效 ── */
  '--yw-ease-ios': 'cubic-bezier(0.32, 0.72, 0, 1)',
  '--yw-ease-spring': 'cubic-bezier(0.34, 1.35, 0.44, 1)',
  '--yw-ease-out': 'cubic-bezier(0.23, 1, 0.32, 1)',
  '--yw-dur-micro': '0.12s',
  '--yw-dur-fast': '0.18s',
  '--yw-dur-ui': '0.25s',
  '--yw-dur-panel': '0.35s',

  /* ── 字体与字号 ── */
  '--yw-font-ui': '-apple-system, BlinkMacSystemFont, "SF Pro Text", Inter, "Segoe UI", "Helvetica Neue", "PingFang SC", "Microsoft YaHei", sans-serif',
  '--yw-fs-menubar': '13px',
  '--yw-fs-title': '13px',
  '--yw-fs-body': '13px',
  '--yw-fs-small': '11px',
}

export const darkTheme: ThemeTokens = {
  '--yw-background': '0.16 0.012 260',
  '--yw-foreground': '0.97 0.003 250',
  '--yw-card': '0.21 0.012 260',
  '--yw-card-foreground': '0.97 0.003 250',
  '--yw-popover': '0.23 0.012 260',
  '--yw-popover-foreground': '0.97 0.003 250',
  '--yw-primary': '0.624 0.206 255.5',
  '--yw-primary-foreground': '0.16 0.012 260',
  '--yw-secondary': '0.28 0.012 260',
  '--yw-secondary-foreground': '0.97 0.003 250',
  '--yw-muted': '0.26 0.01 260',
  '--yw-muted-foreground': '0.7 0.015 260',
  '--yw-accent': '0.3 0.015 260',
  '--yw-accent-foreground': '0.97 0.003 250',
  '--yw-destructive': '0.64 0.19 25',
  '--yw-destructive-foreground': '0.985 0.002 240',
  '--yw-border': '0.32 0.012 260',
  '--yw-input': '0.32 0.012 260',
  '--yw-ring': '0.624 0.206 255.5',
  '--yw-glass': '0.2 0.012 260',
  '--yw-glass-foreground': '0.97 0.003 250',
  '--yw-desktop-icon-label': '0.99 0.002 240',
  '--yw-label-2': 'rgba(255, 255, 255, 0.55)',
  '--yw-label-3': 'rgba(255, 255, 255, 0.3)',
  '--yw-label-4': 'rgba(255, 255, 255, 0.12)',
  '--yw-separator': 'rgba(255, 255, 255, 0.14)',
  '--yw-control-bg': '#3A3A3C',
  '--yw-window-bg': '#2B2B2D',
  '--yw-accent-selection': 'color-mix(in srgb, oklch(var(--yw-primary)) 32%, transparent)',

  '--yw-blur-strong': '30px',
  '--yw-blur-medium': '20px',
  '--yw-blur-light': '12px',
  '--yw-saturate': '180%',

  '--yw-hairline': 'rgba(255, 255, 255, 0.16)',
  '--yw-hairline-weak': 'rgba(255, 255, 255, 0.08)',
  '--yw-specular-top': 'inset 1px 1px 0 rgba(255, 255, 255, 0.18)',
  '--yw-inner-glow': 'inset 0 0 12px rgba(255, 255, 255, 0.06)',

  '--yw-shadow-window': '0 24px 72px rgba(0, 0, 0, 0.6), 0 0 0 0.5px rgba(0, 0, 0, 0.8)',
  '--yw-shadow-window-inactive': '0 10px 34px rgba(0, 0, 0, 0.4), 0 0 0 0.5px rgba(0, 0, 0, 0.6)',
  '--yw-shadow-menu': '0 10px 34px rgba(0, 0, 0, 0.5), 0 0 0 0.5px rgba(255, 255, 255, 0.08)',
  '--yw-shadow-dock': 'inset 1px 1px 0 rgba(255, 255, 255, 0.2), inset 0 0 0 0.5px rgba(255, 255, 255, 0.12), 0 12px 32px rgba(0, 0, 0, 0.5)',
  '--yw-shadow-widget': '0 10px 34px rgba(0, 0, 0, 0.5), 0 0 0 0.5px rgba(255, 255, 255, 0.08)',
  '--yw-shadow-tooltip': '0 6px 20px rgba(0, 0, 0, 0.5), 0 0 0 0.5px rgba(255, 255, 255, 0.1)',

  '--yw-glass-dock-bg': 'rgba(22, 22, 26, 0.55)',
  '--yw-glass-panel-bg': 'rgba(36, 36, 40, 0.72)',
  '--yw-glass-menubar-solid': 'rgba(24, 24, 28, 0.5)',
  '--yw-glass-widget-bg': 'rgba(40, 40, 44, 0.8)',

  '--yw-radius-window': '16px',
  '--yw-radius-sidebar': '12px',
  '--yw-radius-dock': '24px',
  '--yw-radius-dock-icon': '22.37%',
  '--yw-radius-widget': '14px',
  '--yw-radius-menu': '8px',
  '--yw-radius-menu-item': '5px',
  '--yw-radius-spotlight': '20px',
  '--yw-radius-button': '8px',
  '--yw-radius-capsule': '999px',
  '--yw-radius-lg': '16px',
  '--yw-radius-md': '10px',
  '--yw-radius-sm': '6px',

  '--yw-menubar-h': '24px',
  '--yw-titlebar-h': '44px',
  '--yw-tl-size': '12px',
  '--yw-tl-gap': '8px',
  '--yw-tl-inset': '13px',
  '--yw-dock-icon-size': '48px',
  '--yw-dock-pad': '7px',
  '--yw-dock-gap': '4px',
  '--yw-dock-margin': '10px',
  '--yw-deskicon-cell-w': '84px',
  '--yw-deskicon-cell-h': '92px',
  '--yw-deskicon-size': '48px',
  '--yw-resize-hit': '6px',

  '--yw-tl-close': '#FF5F57',
  '--yw-tl-min': '#FEBC2E',
  '--yw-tl-zoom': '#28C840',
  '--yw-tl-inactive': '#565658',

  '--yw-ease-ios': 'cubic-bezier(0.32, 0.72, 0, 1)',
  '--yw-ease-spring': 'cubic-bezier(0.34, 1.35, 0.44, 1)',
  '--yw-ease-out': 'cubic-bezier(0.23, 1, 0.32, 1)',
  '--yw-dur-micro': '0.12s',
  '--yw-dur-fast': '0.18s',
  '--yw-dur-ui': '0.25s',
  '--yw-dur-panel': '0.35s',

  '--yw-font-ui': '-apple-system, BlinkMacSystemFont, "SF Pro Text", Inter, "Segoe UI", "Helvetica Neue", "PingFang SC", "Microsoft YaHei", sans-serif',
  '--yw-fs-menubar': '13px',
  '--yw-fs-title': '13px',
  '--yw-fs-body': '13px',
  '--yw-fs-small': '11px',
}

/** token 合并：自定义主题在亮/暗基础上覆盖（完整 CSS 值或 oklch 通道均可） */
export function mergeTheme(base: ThemeTokens, override?: Partial<ThemeTokens>): ThemeTokens {
  const out: ThemeTokens = { ...base }
  if (override) {
    for (const [k, v] of Object.entries(override)) {
      if (v !== undefined) {
        out[k] = v
      }
    }
  }
  return out
}

/**
 * token → CSS 声明。
 * 语义色（背景/前景/主色等）在 token 表中就是 **oklch 通道三元组**，原样输出；
 * 消费端统一 `oklch(var(--yw-x))` 或带透明度 `oklch(var(--yw-x) / a)`；
 * 实体值（hex/rgba/px/字体等）原样输出，消费端直接 `var(--yw-x)`。
 */
export function tokensToStyle(tokens: ThemeTokens): string {
  return Object.entries(tokens)
    .map(([k, v]) => `${k}: ${v}`)
    .join('; ')
}
