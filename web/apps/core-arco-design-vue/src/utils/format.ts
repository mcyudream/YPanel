// 通用格式化与路径工具（F9：收拢重复实现）。

/** 字节数人性化显示 */
export function fmtBytes(n: number) {
  if (!n) {
    return '0 B'
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v >= 100 || i === 0 ? 0 : 1)} ${units[i]}`
}

/** WS/API 基地址：dev 走 vite 代理，生产同源 */
export function wsBase() {
  return (import.meta.env.DEV && import.meta.env.VITE_ENABLE_PROXY) ? '/proxy' : ''
}
