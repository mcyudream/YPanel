// 加载 iconify 图标
import { downloadAndInstall } from '@/iconify'
import icons from '@/iconify/index.json'
// 自定义指令
import directive from '@/utils/directive'
import { createTerminal } from 'vue-web-terminal'

import App from './App.vue'
import router from './router'
import pinia from './store'
import uiProvider from './ui/provider'
import YdMorphIcon from '@/components/YdMorphIcon/index.vue'
import '@/utils/storage'

// UnoCSS
import 'virtual:uno.css'
// 全局样式
import '@/assets/styles/globals.css'

const app = createApp(App)
app.use(pinia)
app.use(router)
app.use(uiProvider)
// vue-web-terminal 终端核心（YdTerminal 双引擎的 vwt 引擎依赖）
app.use(createTerminal())
directive(app)
// YPanel 主通道图标全局注册（FaIcon 的 yd: 前缀分支依赖此注册）
app.component('YdMorphIcon', YdMorphIcon)
if (icons.isOfflineUse) {
  for (const info of icons.collections) {
    downloadAndInstall(info)
  }
}

app.mount('#app')
