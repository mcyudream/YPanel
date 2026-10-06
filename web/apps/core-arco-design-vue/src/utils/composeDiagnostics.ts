import type * as Monaco from 'monaco-editor'
import Ajv2020 from 'ajv/dist/2020'
import { parse as yamlToJs, parseDocument, type Document } from 'yaml'
// compose-spec 官方 JSON Schema（Apache-2.0）
// https://github.com/compose-spec/compose-spec/blob/master/schema/compose-spec.json
import composeSchema from './compose-spec-schema.json'

// 受管配置的诊断（M23）：compose yaml 语法 + compose-spec 结构校验；daemon.json JSON 语法校验。
// marker 打在 monaco model 上（owner ypanel-managed），保存前可据此阻断语法错误。

const MARK_OWNER = 'ypanel-managed'

export function isComposeYaml(path: string) {
  return path.startsWith('/opt/ypanel/compose/') && /\.ya?ml$/.test(path)
}

export function isDaemonJson(path: string) {
  return path === '/etc/docker/daemon.json'
}

export function isManagedConfig(path: string) {
  return isComposeYaml(path) || isDaemonJson(path)
}

let ajv: Ajv2020 | null = null
let validateFn: ReturnType<Ajv2020['compile']> | null = null

function schemaValidator() {
  if (!validateFn) {
    ajv = new Ajv2020({ allErrors: true, strict: false })
    validateFn = ajv.compile(composeSchema as object)
  }
  return validateFn
}

/** 解析 yaml 文档；语法错误转 marker。 */
function yamlSyntaxMarkers(doc: Document.Parsed, monaco: typeof Monaco, model: Monaco.editor.ITextModel) {
  const markers: Monaco.editor.IMarkerData[] = []
  for (const err of doc.errors) {
    const pos = model.getPositionAt(err.pos?.[0] ?? 0)
    markers.push({
      severity: monaco.MarkerSeverity.Error,
      message: `YAML 语法：${err.message.split('\n')[0]}`,
      startLineNumber: pos.lineNumber,
      startColumn: pos.column,
      endLineNumber: pos.lineNumber,
      endColumn: pos.column + 1,
    })
  }
  return markers
}

/** compose-spec 结构校验；错误尽量定位到对应顶层 key 行。 */
function schemaMarkers(monaco: typeof Monaco, text: string) {
  const validate = schemaValidator()
  let data: unknown
  try {
    data = yamlToJs(text)
  }
  catch {
    return []
  }
  if (!validate(data)) {
    const lines = text.split('\n')
    const markers: Monaco.editor.IMarkerData[] = []
    for (const err of validate.errors || []) {
      // instancePath 如 /services/web/ports → 简化只定位第一段顶层 key 的行
      const topKey = (err.instancePath.split('/').filter(Boolean)[0] || '').toLowerCase()
      let lineNo = 1
      if (topKey) {
        const re = new RegExp(`^(\\s*)${topKey}\\s*:`)
        const hit = lines.findIndex(l => re.test(l))
        if (hit >= 0) {
          lineNo = hit + 1
        }
      }
      markers.push({
        severity: monaco.MarkerSeverity.Warning,
        message: `compose 规范：${err.instancePath || '/'} ${err.message || ''}`,
        startLineNumber: lineNo,
        startColumn: 1,
        endLineNumber: lineNo,
        endColumn: Math.max(2, (lines[lineNo - 1] || ' ').length + 1),
      })
    }
    return markers
  }
  return []
}

function jsonMarkers(monaco: typeof Monaco, model: Monaco.editor.ITextModel, text: string) {
  try {
    JSON.parse(text || '{}')
    return []
  }
  catch (e: any) {
    // JSON.parse 报错带位置（"position N"）
    const m = /position (\d+)/.exec(String(e?.message || ''))
    const pos = model.getPositionAt(m ? Number(m[1]) : 0)
    return [{
      severity: monaco.MarkerSeverity.Error,
      message: `JSON 语法：${e?.message || '解析失败'}`,
      startLineNumber: pos.lineNumber,
      startColumn: pos.column,
      endLineNumber: pos.lineNumber,
      endColumn: pos.column + 1,
    }]
  }
}

/**
 * 刷新受管配置的诊断 marker。
 * @returns 语法级（阻断保存）错误数
 */
export function refreshDiagnostics(monaco: typeof Monaco, model: Monaco.editor.ITextModel, path: string): number {
  if (!isManagedConfig(path)) {
    monaco.editor.setModelMarkers(model, MARK_OWNER, [])
    return 0
  }
  const text = model.getValue()
  let blocking = 0
  const markers: Monaco.editor.IMarkerData[] = []
  if (isDaemonJson(path)) {
    markers.push(...jsonMarkers(monaco, model, text))
  }
  else {
    const doc = parseDocument(text, { strict: false })
    markers.push(...yamlSyntaxMarkers(doc, monaco, model))
    if (!doc.errors.length) {
      markers.push(...schemaMarkers(monaco, text))
    }
  }
  blocking = markers.filter(m => m.severity === monaco.MarkerSeverity.Error).length
  monaco.editor.setModelMarkers(model, MARK_OWNER, markers)
  return blocking
}
