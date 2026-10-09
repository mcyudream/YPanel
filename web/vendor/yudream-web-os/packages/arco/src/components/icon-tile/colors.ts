/**
 * 图标底座色板：macOS 应用图标风格的渐变（蓝/紫/粉/橙/绿/青/石墨/红）。
 * 按 key hash 稳定选色——同一应用在任何位置（桌面/Dock/启动台）颜色一致。
 */
const TILE_GRADIENTS = [
  'linear-gradient(135deg, #41A8F8 0%, #0B63D8 100%)',
  'linear-gradient(135deg, #B470F2 0%, #7B3FD4 100%)',
  'linear-gradient(135deg, #FF74B7 0%, #E93A8C 100%)',
  'linear-gradient(135deg, #FFAD52 0%, #F2701D 100%)',
  'linear-gradient(135deg, #6FD66F 0%, #2FA341 100%)',
  'linear-gradient(135deg, #4ED4D4 0%, #1D9FA8 100%)',
  'linear-gradient(135deg, #FF7A73 0%, #DE342C 100%)',
  'linear-gradient(135deg, #9A9AA5 0%, #5C5C66 100%)',
]

/** 稳定 hash（djb2 变体） */
function hashKey(key: string): number {
  let h = 5381
  for (let i = 0; i < key.length; i++) {
    h = ((h << 5) + h + key.charCodeAt(i)) | 0
  }
  return Math.abs(h)
}

/** 按应用 key 取底座渐变；manifest 显式给了 iconBg 则优先 */
export function tileBackground(key: string, iconBg?: string): string {
  if (iconBg) {
    return iconBg
  }
  return TILE_GRADIENTS[hashKey(key) % TILE_GRADIENTS.length]!
}
