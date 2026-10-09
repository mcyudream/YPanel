// 加载 iconify 图标
import { downloadAndInstall } from '@/iconify'
import icons from '@/iconify/index.json'
// 自定义指令
import directive from '@/utils/directive'

import { createWebOS } from '@yudream/yudream-webos-vue'
import { arcoAdapter } from '@yudream/yudream-webos-arco'
import App from './App.vue'
import router from './router'
import pinia from './store'
import uiProvider from './ui/provider'
import YdMorphIcon from '@/components/YdMorphIcon/index.vue'
import { i18n, initLocale } from '@/locales'
import '@/utils/storage'

// UnoCSS
import 'virtual:uno.css'
// 全局样式
import '@/assets/styles/globals.css'

const app = createApp(App)
app.use(pinia)
app.use(router)
app.use(uiProvider)
app.use(i18n)
initLocale()
// 桌面工作台（/desktop）基于 YudreamWebOS：应用在路由壳内按需注册，
// 经典模式常驻本插件实例仅为提供注入上下文（构造为纯 TS 服务，无 DOM 副作用）
app.use(createWebOS({
  ui: arcoAdapter,
  persist: { adapter: 'localstorage', prefix: 'ypanel.webos' },
  desktop: { arrangeMode: 'grid', collision: 'swap' },
  dock: { position: 'bottom', magnification: true },
  // 小组件直接铺在桌面（按各自尺寸自适应），非侧栏列
  widgets: { placement: 'desktop' },
  menubar: { showControlCenter: true },
  windows: { snapToEdge: true, sessionRestore: true },
}))
directive(app)
// YPanel 主通道图标全局注册（FaIcon 的 yd: 前缀分支依赖此注册）
app.component('YdMorphIcon', YdMorphIcon)
if (icons.isOfflineUse) {
  for (const info of icons.collections) {
    downloadAndInstall(info)
  }
}

app.mount('#app')
