import { defineStore } from 'pinia'

// 通知中心全局弹窗状态：顶栏铃铛「查看全部」/ 概览页均可唤起（不跳页面）。
export const useNotificationCenterStore = defineStore('notificationCenter', {
  state: () => ({
    visible: false,
    refreshTick: 0,
  }),
  actions: {
    open() {
      this.visible = true
      this.refreshTick++
    },
    close() {
      this.visible = false
    },
    refresh() {
      this.refreshTick++
    },
  },
})
