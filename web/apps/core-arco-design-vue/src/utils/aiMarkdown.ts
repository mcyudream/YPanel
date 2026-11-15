import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { loadMonaco, monacoThemeName } from './monacoLoader'

interface MarkdownBlock { code?: string, language: string, html: string }
const markdownCache = new Map<string, MarkdownBlock[]>()
const codeCache = new Map<string, Promise<string>>()

export function aiMarkdown(text: string, cache: boolean): MarkdownBlock[] {
  try {
    return parseMarkdown(text, cache)
  }
  catch {
    return [{ language: '', html: text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/\n/g, '<br>') }]
  }
}

function parseMarkdown(text: string, cache: boolean) {
  const existing = markdownCache.get(text)
  if (existing) {
    return existing
  }
  const tokens = marked.lexer(text)
  const blocks: MarkdownBlock[] = tokens.map((token) => {
    if (token.type === 'code') {
      return { code: token.text, language: token.lang?.split(/\s/)[0] || 'text', html: '' }
    }
    const list = Object.assign([token], { links: tokens.links })
    return { language: '', html: DOMPurify.sanitize(marked.parser(list, { breaks: true }), { USE_PROFILES: { html: true }, FORBID_TAGS: ['img', 'style', 'input', 'button', 'form'], FORBID_ATTR: ['style', 'id', 'name'] }) }
  })
  if (cache && text.length < 200000) {
    markdownCache.set(text, blocks)
    if (markdownCache.size > 128) {
      markdownCache.delete(markdownCache.keys().next().value!)
    }
  }
  return blocks
}

export async function aiHighlight(code: string, language: string, dark: boolean) {
  const key = `${dark}\0${language}\0${code}`
  let result = codeCache.get(key)
  if (!result) {
    result = loadMonaco().then(async (monaco) => {
      monaco.editor.setTheme(monacoThemeName(dark ? 'dark' : 'light'))
      return DOMPurify.sanitize(await monaco.editor.colorize(code, language, {}), { ALLOWED_TAGS: ['span', 'br'], ALLOWED_ATTR: ['class'] })
    })
    codeCache.set(key, result)
    if (codeCache.size > 128) {
      codeCache.delete(codeCache.keys().next().value!)
    }
  }
  return result
}
