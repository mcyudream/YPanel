import { ref } from 'vue'
import type { AppTask } from '@/api/modules/task'
import { taskApi } from '@/api/modules/task'
import { useTaskCenterStore } from '@/store/modules/taskCenter'

/**
 * useTaskJob 耗时任务提交统一交互（M65）：
 * submit() 提交接口拿 { taskId } → 自动打开任务中心日志视图（实时滚动）→
 * 2s 轮询任务态到终态 → 成功/失败回调并解除 running。
 * 返回 submit 的完整响应（如镜像导出附带产物路径）。
 */
export function useTaskJob() {
  const running = ref(false)

  async function run<T extends { taskId: number }>(
    submit: () => Promise<T>,
    opts: {
      // 提交后自动打开任务中心日志视图（默认 true；多个任务连续提交时可关）
      openLog?: boolean
      onSuccess?: (task: AppTask, out: T) => void | Promise<void>
      onError?: (task: AppTask) => void
    } = {},
  ): Promise<T | null> {
    if (running.value) {
      return null
    }
    const toast = useFaToast()
    running.value = true
    let out: T
    try {
      out = await submit()
    }
    catch (e: any) {
      running.value = false
      toast.error(e?.message || '提交失败')
      return null
    }
    const taskId = out.taskId
    if (opts.openLog !== false) {
      useTaskCenterStore().open(taskId)
    }
    const poll = setInterval(async () => {
      let t: AppTask
      try {
        t = await taskApi.get(taskId)
      }
      catch {
        return // 单次轮询失败下一轮重试
      }
      if (t.status === 'running') {
        return
      }
      clearInterval(poll)
      running.value = false
      if (t.status === 'success') {
        await opts.onSuccess?.(t, out)
      }
      else {
        toast.error(t.error || '任务执行失败')
        opts.onError?.(t)
      }
    }, 2000)
    return out
  }

  return { running, run }
}
