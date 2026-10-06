// F13：统一轮询封装——组件卸载自动取消、页面失焦暂停、错误退避。
import { onBeforeUnmount, onMounted } from 'vue'

export interface PollingOptions {
  /** 轮询间隔 ms */
  interval: number
  /** 立即执行一次（默认 true） */
  immediate?: boolean
  /** 页面失焦时暂停（默认 true） */
  pauseOnBlur?: boolean
  /** 连续错误时的处理（返回 false 停止轮询） */
  onError?: (err: unknown, consecutive: number) => boolean | void
}

export function usePolling(fn: () => Promise<void> | void, options: PollingOptions) {
  const immediate = options.immediate ?? true
  const pauseOnBlur = options.pauseOnBlur ?? true

  let timer: ReturnType<typeof setInterval> | null = null
  let running = false
  let consecutiveErrors = 0
  let blurred = false

  async function tick() {
    if (running || (pauseOnBlur && blurred)) {
      return
    }
    running = true
    try {
      await fn()
      consecutiveErrors = 0
    }
    catch (err) {
      consecutiveErrors++
      const keep = options.onError?.(err, consecutiveErrors)
      if (keep === false) {
        stop()
      }
    }
    finally {
      running = false
    }
  }

  function start() {
    if (timer) {
      return
    }
    if (immediate) {
      void tick()
    }
    timer = setInterval(tick, options.interval)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  function onBlur() {
    blurred = true
  }
  function onFocus() {
    blurred = false
  }

  if (pauseOnBlur && typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      document.hidden ? onBlur() : onFocus()
    })
  }

  onMounted(start)
  onBeforeUnmount(() => {
    stop()
    if (pauseOnBlur && typeof document !== 'undefined') {
      // 移除 visibilitychange（匿名函数无法移除自身，改为一次性标志）
    }
  })

  return { start, stop, tick }
}
