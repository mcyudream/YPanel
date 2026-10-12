// 剪贴板写入统一封装：优先 async Clipboard API（安全上下文），
// http 内网 IP 等非安全上下文下 navigator.clipboard 为 undefined，
// 回落 execCommand('copy')（须在用户手势调用栈内同步执行才生效）。
export async function copyText(text: string): Promise<boolean> {
  if (!text) {
    return false
  }
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  }
  catch {}
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  }
  catch {}
  ta.remove()
  return ok
}

// 剪贴板读取：仅安全上下文可用，非安全上下文/被拒时返回 null（调用方走手动粘贴降级）
export async function readClipboardText(): Promise<string | null> {
  try {
    if (navigator.clipboard?.readText) {
      return await navigator.clipboard.readText()
    }
  }
  catch {}
  return null
}
