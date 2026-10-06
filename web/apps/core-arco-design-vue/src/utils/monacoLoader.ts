import type * as Monaco from 'monaco-editor'

export type MonacoNamespace = typeof Monaco

let monacoPromise: Promise<MonacoNamespace> | null = null

// fa 主题近似色（monaco 主题不支持 CSS 变量，取 fa 明暗两套底色）
const THEMES: Record<'light' | 'dark', Monaco.editor.IStandaloneThemeData> = {
  light: {
    base: 'vs',
    inherit: true,
    rules: [],
    colors: {
      'editor.background': '#ffffff',
      'editorLineNumber.foreground': '#a1a1aa',
      'editorLineNumber.activeForeground': '#52525b',
      'editorGutter.background': '#ffffff',
      'editor.lineHighlightBackground': '#f4f4f580',
      'editorIndentGuide.background1': '#e4e4e7',
      'editorWidget.background': '#ffffff',
      'editorWidget.border': '#e4e4e7',
      'scrollbarSlider.background': '#a1a1aa40',
    },
  },
  dark: {
    base: 'vs-dark',
    inherit: true,
    rules: [],
    colors: {
      'editor.background': '#1c1c1a',
      'editorLineNumber.foreground': '#57534e',
      'editorLineNumber.activeForeground': '#a8a29e',
      'editorGutter.background': '#1c1c1a',
      'editor.lineHighlightBackground': '#29252466',
      'editorIndentGuide.background1': '#292524',
      'editorWidget.background': '#1c1c1a',
      'editorWidget.border': '#44403c',
      'scrollbarSlider.background': '#57534e66',
    },
  },
}

export function monacoThemeName(colorScheme: string) {
  return colorScheme === 'dark' ? 'ypanel-dark' : 'ypanel-light'
}

export function loadMonaco(): Promise<MonacoNamespace> {
  if (!monacoPromise) {
    monacoPromise = (async () => {
      self.MonacoEnvironment = {
        getWorker(_workerId: string, label: string) {
          // 字面量 new URL：vite 静态分析产出 worker chunk（rolldown 下 ?worker 深导入不可用）
          switch (label) {
            case 'json':
              return new Worker(new URL('./monaco-workers/json.js', import.meta.url), { type: 'module' })
            case 'css':
            case 'scss':
            case 'less':
              return new Worker(new URL('./monaco-workers/css.js', import.meta.url), { type: 'module' })
            case 'html':
            case 'handlebars':
            case 'razor':
              return new Worker(new URL('./monaco-workers/html.js', import.meta.url), { type: 'module' })
            case 'typescript':
            case 'javascript':
              return new Worker(new URL('./monaco-workers/typescript.js', import.meta.url), { type: 'module' })
            default:
              return new Worker(new URL('./monaco-workers/editor.js', import.meta.url), { type: 'module' })
          }
        },
      }
      const monaco = await import('monaco-editor')
      monaco.editor.defineTheme('ypanel-light', THEMES.light)
      monaco.editor.defineTheme('ypanel-dark', THEMES.dark)
      return monaco
    })()
  }
  return monacoPromise
}
