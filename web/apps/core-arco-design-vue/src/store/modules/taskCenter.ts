import { defineStore } from 'pinia'

// 任务中心全局弹窗状态：任意页面（顶栏图标 / 安装向导 / 镜像拉取）均可唤起。
export const useTaskCenterStore = defineStore('taskCenter', {
  state: () => ({
    visible: false,
    // 打开时直接定位到某任务日志（0 = 列表态）
    activeTaskId: 0,
    // 刷新信号（递增）：弹窗内列表收到后重新拉取
    refreshTick: 0,
  }),
  actions: {
    open(taskId = 0) {
      this.activeTaskId = taskId
      this.visible = true
      this.refreshTick++
    },
    close() {
      this.visible = false
      this.activeTaskId = 0
    },
    refresh() {
      this.refreshTick++
    },
  },
})
