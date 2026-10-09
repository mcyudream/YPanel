// 桌面内置壁纸清单：随前端资产打包（go:embed 单二进制内嵌，无外链依赖）。
// 来源 Unsplash（Unsplash License，可免费商用）；2400px q78。
// name 走 i18n（desktop.wallpaper.*），访问时求值（非组件模块，统一 i18n.global.t）。
import { i18n } from '@/locales'
import cloudPeaks from '@/assets/wallpapers/cloud-peaks.jpg'
import darkDunes from '@/assets/wallpapers/dark-dunes.jpg'
import milkywayPeaks from '@/assets/wallpapers/milkyway-peaks.jpg'
import nightSky from '@/assets/wallpapers/night-sky.jpg'
import purpleRidge from '@/assets/wallpapers/purple-ridge.jpg'

export interface WallpaperItem {
  id: string
  name: string
  src: string
}

export const wallpapers: WallpaperItem[] = [
  { id: 'milkyway-peaks', get name() { return i18n.global.t('desktop.wallpaper.milkywayPeaks') }, src: milkywayPeaks },
  { id: 'cloud-peaks', get name() { return i18n.global.t('desktop.wallpaper.cloudPeaks') }, src: cloudPeaks },
  { id: 'purple-ridge', get name() { return i18n.global.t('desktop.wallpaper.purpleRidge') }, src: purpleRidge },
  { id: 'dark-dunes', get name() { return i18n.global.t('desktop.wallpaper.darkDunes') }, src: darkDunes },
  { id: 'night-sky', get name() { return i18n.global.t('desktop.wallpaper.nightSky') }, src: nightSky },
]
