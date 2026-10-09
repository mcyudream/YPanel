/**
 * 颜色工具：hex → oklch 通道三元组（"L C H"）。
 * 用途：Accent Color 接口允许宿主传 hex，但语义 token 需要保持 oklch 通道格式，
 * 这样消费端 oklch(var(--yw-primary) / alpha) 在两种来源下都有效。
 */

function srgbToLinear(c: number): number {
  return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

/** hex (#RGB/#RRGGBB) → oklch 通道 "L C H"，L/C/H 各自四舍五入到 3/3/1 位 */
export function hexToOklchChannels(hex: string): string {
  let h = hex.trim().replace(/^#/, '')
  if (h.length === 3) {
    h = h.split('').map(c => c + c).join('')
  }
  if (!/^[0-9a-f]{6}$/i.test(h)) {
    return hex // 非法输入原样返回（CSS 端保持原值）
  }
  const r = srgbToLinear(Number.parseInt(h.slice(0, 2), 16) / 255)
  const g = srgbToLinear(Number.parseInt(h.slice(2, 4), 16) / 255)
  const b = srgbToLinear(Number.parseInt(h.slice(4, 6), 16) / 255)

  // lin sRGB → OKLab（Björn Ottosson 标准矩阵）
  const l = 0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b
  const m = 0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b
  const s = 0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b
  const l_ = Math.cbrt(l)
  const m_ = Math.cbrt(m)
  const s_ = Math.cbrt(s)

  const labL = 0.2104542553 * l_ + 0.793617785 * m_ - 0.0040720468 * s_
  const labA = 1.9779984951 * l_ - 2.428592205 * m_ + 0.4505937099 * s_
  const labB = 0.0259040371 * l_ + 0.7827717662 * m_ - 0.808675766 * s_

  const C = Math.sqrt(labA * labA + labB * labB)
  let H = Math.atan2(labB, labA) * 180 / Math.PI
  if (H < 0) {
    H += 360
  }

  return `${labL.toFixed(3)} ${C.toFixed(3)} ${H.toFixed(1)}`
}
