import type * as Monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker.js?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker.js?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker.js?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker.js?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker.js?worker'

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
          switch (label) {
            case 'json':
              return new jsonWorker()
            case 'css':
            case 'scss':
            case 'less':
              return new cssWorker()
            case 'html':
            case 'handlebars':
            case 'razor':
              return new htmlWorker()
            case 'typescript':
            case 'javascript':
              return new tsWorker()
            default:
              return new editorWorker()
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
