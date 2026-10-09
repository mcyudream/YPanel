/**
 * 样式入口：宿主 import '@yudream/yudream-webos-vue/styles' 引入。
 */
import { builtinAdapterCss } from './styles/builtin-css'

let injected = false

/** 注入内置适配器样式（幂等） */
export function injectBuiltinStyles() {
  if (injected || typeof document === 'undefined') {
    return
  }
  injected = true
  const el = document.createElement('style')
  el.setAttribute('data-yw-builtin-styles', '')
  el.textContent = builtinAdapterCss
  document.head.appendChild(el)
}

injectBuiltinStyles()

export { builtinAdapterCss }
