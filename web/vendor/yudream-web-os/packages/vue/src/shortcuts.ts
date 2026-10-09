/**
 * 全局快捷键注册表：metaKey || ctrlKey 双映射（Web 环境拿不到真 Cmd）。
 * 冲突时后者覆盖前者并告警。
 */
type Handler = (e: KeyboardEvent) => void

const bindings = new Map<string, Handler>()

/** 规范化：'Cmd/Ctrl+Space' → 'mod+space' */
function normalize(combo: string): string {
  const parts = combo
    .toLowerCase()
    .replace(/cmd\/ctrl|cmd\+ctrl|meta\/ctrl|mod/g, 'mod')
    .replace(/cmd|meta|ctrl/g, 'mod')
    .split('+')
    .map(k => k.trim())
    .filter(Boolean)
  // 与 eventCombo 的构建顺序一致（mod → shift → alt → key），否则注册与触发组合串对不上
  const out: string[] = []
  for (const key of ['mod', 'shift', 'alt']) {
    const i = parts.indexOf(key)
    if (i > -1) {
      out.push(key)
    }
  }
  for (const k of parts) {
    if (!out.includes(k)) {
      out.push(k)
    }
  }
  return out.join('+')
}

function eventCombo(e: KeyboardEvent): string {
  const parts: string[] = []
  if (e.metaKey || e.ctrlKey) {
    parts.push('mod')
  }
  if (e.shiftKey) {
    parts.push('shift')
  }
  if (e.altKey) {
    parts.push('alt')
  }
  parts.push(e.key.toLowerCase())
  return parts.join('+')
}

export const shortcuts = {
  /** 注册快捷键；冲突时覆盖并告警 */
  register(combo: string, handler: Handler): () => void {
    const key = normalize(combo)
    if (bindings.has(key)) {
      console.warn(`[webos:shortcuts] "${combo}" 覆盖了已注册的快捷键`)
    }
    bindings.set(key, handler)
    return () => bindings.delete(key)
  },
  unregister(combo: string): void {
    bindings.delete(normalize(combo))
  },
  has(combo: string): boolean {
    return bindings.has(normalize(combo))
  },
  /** window keydown 分发（provider 安装时挂载） */
  dispatch(e: KeyboardEvent): boolean {
    const handler = bindings.get(eventCombo(e))
    if (handler) {
      handler(e)
      return true
    }
    return false
  },
  attach(target: Window | HTMLElement = window): () => void {
    const onKeydown = (e: KeyboardEvent) => {
      if (shortcuts.dispatch(e)) {
        e.preventDefault()
      }
    }
    target.addEventListener('keydown', onKeydown as EventListener)
    return () => target.removeEventListener('keydown', onKeydown as EventListener)
  },
}
