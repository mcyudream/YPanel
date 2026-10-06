// 文件编码检测/解码（与 agent 端 x/text 白名单对齐）。
// 检测顺序：UTF-8 严格 → GB18030 → Big5 → Shift-JIS → Windows-1252（均严格，失败落兜底）。

export interface FileEncodingOption {
  id: string
  label: string
}

export const FILE_ENCODINGS: FileEncodingOption[] = [
  { id: 'utf-8', label: 'UTF-8' },
  { id: 'gb18030', label: 'GB18030（GBK 兼容）' },
  { id: 'big5', label: 'Big5（繁体中文）' },
  { id: 'shift_jis', label: 'Shift-JIS（日文）' },
  { id: 'windows-1252', label: 'Windows-1252（西文）' },
]

export function encodingLabel(id: string) {
  return FILE_ENCODINGS.find(e => e.id === id)?.label ?? id
}

// base64 → 字节
export function b64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) {
    bytes[i] = bin.charCodeAt(i)
  }
  return bytes
}

// 字节 → base64（写盘不再走此路径，仅供调试）
export function bytesToB64(bytes: Uint8Array): string {
  let bin = ''
  for (let i = 0; i < bytes.length; i++) {
    bin += String.fromCharCode(bytes[i])
  }
  return btoa(bin)
}

// 用指定编码解码；TextDecoder 标签与 agent 白名单一致
export function decodeWith(bytes: Uint8Array, encodingId: string): string {
  return new TextDecoder(encodingId === 'utf-8' ? 'utf-8' : encodingId).decode(bytes)
}

// 依次尝试严格解码，返回首个成功的编码与文本
export function detectEncoding(bytes: Uint8Array): { encoding: string, text: string } {
  const candidates = ['utf-8', 'gb18030', 'big5', 'shift_jis', 'windows-1252']
  for (const enc of candidates) {
    try {
      const text = new TextDecoder(enc, { fatal: true }).decode(bytes)
      return { encoding: enc, text }
    }
    catch {}
  }
  // 兜底：windows-1252 无非法字节，理论上不会到这里
  return { encoding: 'windows-1252', text: decodeWith(bytes, 'windows-1252') }
}

// 探测换行符：文本含 \r\n 即 CRLF
export function detectEol(text: string): 'lf' | 'crlf' {
  return text.includes('\r\n') ? 'crlf' : 'lf'
}

// 扩展名 → monaco 语言 id（monaco 内置映射之外的常见补充）
const EXT_LANG: Record<string, string> = {
  go: 'go', js: 'javascript', mjs: 'javascript', cjs: 'javascript', ts: 'typescript', tsx: 'typescript', jsx: 'javascript',
  vue: 'html', html: 'html', htm: 'html', xml: 'xml', svg: 'xml', json: 'json', jsonc: 'json', yml: 'yaml', yaml: 'yaml',
  css: 'css', scss: 'scss', less: 'less', md: 'markdown', markdown: 'markdown', sql: 'sql', py: 'python', rb: 'ruby',
  php: 'php', sh: 'shell', bash: 'shell', zsh: 'shell', conf: 'ini', ini: 'ini', env: 'ini', toml: 'ini', properties: 'ini',
  java: 'java', c: 'c', h: 'c', cpp: 'cpp', hpp: 'cpp', rs: 'rust', kt: 'kotlin', cs: 'csharp', dockerfile: 'dockerfile',
  lua: 'lua', pl: 'perl', bat: 'bat', ps1: 'powershell', log: 'log', txt: 'plaintext',
}

export function languageOf(path: string): string {
  const name = path.slice(path.lastIndexOf('/') + 1).toLowerCase()
  if (name === 'dockerfile' || name.endsWith('.dockerfile')) {
    return 'dockerfile'
  }
  if (name === 'makefile') {
    return 'shell'
  }
  const ext = name.includes('.') ? name.slice(name.lastIndexOf('.') + 1) : ''
  return EXT_LANG[ext] ?? 'plaintext'
}
