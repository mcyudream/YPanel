// 生成 YPanel 本地图标数据（构建期脚本，产物随 git 提交，前端零外链）。
// 数据源：lucide-static（ISC）的 icon-nodes.json + tags.json → src/ui/icons/data.json
// 图标节点格式与 morphicons 的 IconNode 直接兼容（24×24 描边集）。
// 运行：pnpm --filter @fantastic-admin/core-arco-design-vue exec node ../../scripts/gen-icons.mjs
//      （或在 web/ 下 node scripts/gen-icons.mjs）
import { createRequire } from 'node:module'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const require = createRequire(import.meta.url)
const __dirname = path.dirname(fileURLToPath(import.meta.url))

const nodes = require(require.resolve('lucide-static/icon-nodes.json', { paths: [path.resolve(__dirname, '../apps/core-arco-design-vue')] }))
const tags = require(require.resolve('lucide-static/tags.json', { paths: [path.resolve(__dirname, '../apps/core-arco-design-vue')] }))

// 分类推断：lucide tags 是关键词数组，用关键词首词聚类太散；改为内置分类映射 + 关键词搜索
// （选择器的“分类”按 YPanel 场景组织，命中关键词即归类）
const categories = {
  服务器: ['server', 'cpu', 'hard-drive', 'memory', 'database', 'globe', 'network', 'cloud', 'router', 'monitor'],
  文件: ['folder', 'file', 'save', 'archive', 'copy', 'download', 'upload', 'clipboard'],
  容器与部署: ['box', 'container', 'package', 'ship', 'rocket', 'layers', 'stack', 'puzzle', 'blocks'],
  系统操作: ['power', 'settings', 'refresh', 'rotate', 'wrench', 'hammer', 'terminal', 'code', 'play', 'square', 'pause', 'circle-stop'],
  安全: ['lock', 'key', 'shield', 'fingerprint', 'eye', 'user', 'users', 'log-in', 'log-out'],
  状态: ['check', 'circle-alert', 'circle-x', 'info', 'triangle-alert', 'activity', 'gauge', 'signal', 'wifi', 'zap', 'flame'],
  导航: ['chevron', 'arrow', 'menu', 'list', 'grid', 'layout', 'panel', 'sidebar'],
  编辑: ['pencil', 'pen', 'edit', 'trash', 'plus', 'minus', 'scissors', 'type', 'text'],
  媒体: ['image', 'camera', 'video', 'music', 'volume', 'mic', 'bell'],
  时间: ['clock', 'calendar', 'timer', 'hourglass', 'history', 'watch'],
}

function categorize(name) {
  const hit = []
  for (const [cat, keywords] of Object.entries(categories)) {
    for (const kw of keywords) {
      if (name === kw || name.startsWith(`${kw}-`) || name.includes(kw)) {
        hit.push(cat)
        break
      }
    }
  }
  return hit
}

const out = { version: 1, source: 'lucide', icons: {} }
const names = Object.keys(nodes).sort()
for (const name of names) {
  const tagList = Array.isArray(tags[name]) ? tags[name] : []
  out.icons[name] = {
    tags: tagList,
    categories: categorize(name),
    nodes: nodes[name],
  }
}

const targetDir = path.resolve(__dirname, '../apps/core-arco-design-vue/src/ui/icons')
fs.mkdirSync(targetDir, { recursive: true })
fs.writeFileSync(path.join(targetDir, 'data.json'), JSON.stringify(out))
console.log(`[gen-icons] ${names.length} icons -> src/ui/icons/data.json (${(fs.statSync(path.join(targetDir, 'data.json')).size / 1024 / 1024).toFixed(2)} MB)`)
