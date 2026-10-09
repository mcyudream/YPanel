import { loadingFadeOut } from 'virtual:app-loading'
import { createRouter, createWebHashHistory, createWebHistory } from 'vue-router'
import pinia from '@/store'
import setupExtensions from './extensions'
import setupGuards from './guards'
// 路由相关数据
import { constantRoutes } from './routes'

const router = createRouter({
  history: useAppSettingsStore(pinia).settings.app.routeMode === 'hash' ? createWebHashHistory() : createWebHistory(),
  routes: constantRoutes,
  strict: true,
})

setupGuards(router)
setupExtensions(router)

// M49 入口路径清洗：安全入口模式下，登录成功后面板各页 URL 不携带入口段。
// 未登录时入口路径保留（/入口 → 登录页）；登录后每次导航把 /入口#/x 重写为 /#/x。
router.afterEach(() => {
  if (localStorage.getItem('token') && location.pathname !== '/') {
    window.history.replaceState(null, '', '/' + location.hash)
  }
})

router.isReady().then(() => {
  loadingFadeOut()
})

export default router
