/** 主题 token（完整 CSS 值或 oklch 通道，消费端直接使用） */
export type ThemeTokens = Record<string, string>

/** 主题模式 */
export type ThemeMode = 'light' | 'dark' | 'system'

/** 毛玻璃档位 */
export type GlassLevel = 'none' | 'sm' | 'md' | 'lg' | 'xl'

export interface WallpaperMeta {
  /** 壁纸 URL（图片）、渐变 CSS 值 */
  src: string
  fit?: 'cover' | 'contain' | 'tile'
  dimInDark?: boolean
}

/** 毛玻璃模糊半径（px）与透明度档位映射 */
export const GLASS_LEVELS: Record<GlassLevel, { blur: number, alpha: number }> = {
  none: { blur: 0, alpha: 1 },
  sm: { blur: 8, alpha: 0.85 },
  md: { blur: 16, alpha: 0.75 },
  lg: { blur: 24, alpha: 0.65 },
  xl: { blur: 40, alpha: 0.55 },
}
