<script setup lang="ts">
import type { ComposeTopologyEdge, ComposeTopologyNode } from '@/api/modules/compose'
import { i18n } from '@/locales'

// M26 P1：compose 项目服务拓扑（分层布局 + SVG 连线）。
// 节点为 HTML（随主题/暗色自适应），连线为底层 SVG；只读展示，交互（操作服务）由宿主处理。
const props = defineProps<{
  nodes: ComposeTopologyNode[]
  edges: ComposeTopologyEdge[]
}>()

const emit = defineEmits<{
  (e: 'node-click', name: string): void
}>()

const NODE_W = 176
const NODE_H = 62
const GAP_X = 88
const GAP_Y = 28

interface Placed {
  node: ComposeTopologyNode
  x: number
  y: number
}

// 层号 = 依赖链深度：无依赖者为 0 层（最左），依赖者在被依赖者右侧 +1。
// Kahn 拓扑排序；成环节点兜底放到最后一层，保证全部可见。
const layout = computed(() => {
  const byName = new Map(props.nodes.map(n => [n.name, n]))
  const deps = new Map<string, string[]>()
  const dependents = new Map<string, string[]>()
  for (const n of props.nodes) {
    deps.set(n.name, [])
    dependents.set(n.name, [])
  }
  for (const e of props.edges) {
    if (e.from === e.to || !byName.has(e.from) || !byName.has(e.to)) {
      continue
    }
    deps.get(e.from)!.push(e.to)
    dependents.get(e.to)!.push(e.from)
  }

  const layer = new Map<string, number>()
  const remaining = new Map<string, number>()
  const queue: string[] = []
  for (const n of props.nodes) {
    const d = deps.get(n.name)!.length
    remaining.set(n.name, d)
    if (d === 0) {
      layer.set(n.name, 0)
      queue.push(n.name)
    }
  }
  let maxLayer = 0
  for (let i = 0; i < queue.length; i++) {
    const u = queue[i]!
    const lu = layer.get(u)!
    maxLayer = Math.max(maxLayer, lu)
    for (const v of dependents.get(u)!) {
      layer.set(v, Math.max(layer.get(v) ?? 0, lu + 1))
      maxLayer = Math.max(maxLayer, layer.get(v)!)
      const r = remaining.get(v)! - 1
      remaining.set(v, r)
      if (r === 0) {
        queue.push(v)
      }
    }
  }
  // 环上节点（Kahn 未覆盖）兜底
  for (const n of props.nodes) {
    if (!layer.has(n.name)) {
      layer.set(n.name, maxLayer + 1)
    }
  }

  const cols = new Map<number, string[]>()
  for (const n of props.nodes) {
    const l = layer.get(n.name)!
    if (!cols.has(l)) {
      cols.set(l, [])
    }
    cols.get(l)!.push(n.name)
  }
  for (const list of cols.values()) {
    list.sort()
  }

  const colCount = Math.max(...cols.keys(), 0) + 1
  const maxRows = Math.max(1, ...[...cols.values()].map(a => a.length))
  const height = maxRows * NODE_H + (maxRows - 1) * GAP_Y
  const width = colCount * NODE_W + (colCount - 1) * GAP_X

  const pos = new Map<string, Placed>()
  for (const [l, names] of cols) {
    const colH = names.length * NODE_H + (names.length - 1) * GAP_Y
    const offset = Math.max(0, (height - colH) / 2)
    names.forEach((name, i) => {
      pos.set(name, { node: byName.get(name)!, x: l * (NODE_W + GAP_X), y: offset + i * (NODE_H + GAP_Y) })
    })
  }
  return { pos, width, height, layer }
})

const placedList = computed(() => [...layout.value.pos.entries()].map(([name, p]) => ({ name, ...p })))

// 连线：从被依赖者（layer 小，左）出发指向依赖方（右），箭头落在依赖方入口。
// 线型按依赖语义与被依赖者健康：healthy 实线 / unhealthy 红 / completed（一次性初始化）虚线 / 其余实线灰。
interface EdgePath {
  d: string
  cls: string
  title: string
}

const edgePaths = computed<EdgePath[]>(() => {
  const out: EdgePath[] = []
  for (const e of props.edges) {
    const a = layout.value.pos.get(e.to)
    const b = layout.value.pos.get(e.from)
    if (!a || !b) {
      continue
    }
    const x0 = a.x + NODE_W
    const y0 = a.y + NODE_H / 2
    const x1 = b.x
    const y1 = b.y + NODE_H / 2
    const dx = Math.max(36, (x1 - x0) / 2)
    const target = props.nodes.find(n => n.name === e.to)
    const unhealthy = target?.state && target.state !== 'running'
    const cls = e.condition === 'service_completed_successfully' || unhealthy
      ? 'topo-edge-dash'
      : e.condition === 'service_healthy' && target?.health === 'unhealthy'
        ? 'topo-edge-bad'
        : 'topo-edge'
    out.push({
      d: `M ${x0} ${y0} C ${x0 + dx} ${y0}, ${x1 - dx} ${y1}, ${x1} ${y1}`,
      cls,
      title: i18n.global.t('components.ydTopology.edgeDepends', { from: e.from, to: e.to, cond: e.condition || 'service_started' }),
    })
  }
  return out
})

const stateCls: Record<string, string> = {
  running: 'border-emerald-500/50',
  exited: 'border-muted',
  created: 'border-blue-500/40',
  paused: 'border-amber-500/50',
  restarting: 'border-amber-500/50',
  dead: 'border-red-500/60',
}

const dotCls: Record<string, string> = {
  running: 'bg-emerald-500 animate-pulse',
  exited: 'bg-muted-foreground/40',
  created: 'bg-blue-500',
  paused: 'bg-amber-500',
  restarting: 'bg-amber-500 animate-pulse',
  dead: 'bg-red-500',
}

function badgeCls(n: ComposeTopologyNode) {
  if (n.health === 'healthy') {
    return 'bg-emerald-500/10 text-emerald-600'
  }
  if (n.health === 'unhealthy') {
    return 'bg-red-500/10 text-red-600'
  }
  if (n.health === 'starting') {
    return 'bg-amber-500/10 text-amber-600'
  }
  return ''
}

function badgeText(n: ComposeTopologyNode) {
  if (n.health) {
    return n.health
  }
  return n.state || i18n.global.t('components.ydTopology.stateNone')
}

function onNode(name: string) {
  emit('node-click', name)
}
</script>

<template>
  <div class="topo-wrap overflow-auto rounded-md bg-muted/20">
    <div class="relative" :style="{ width: `${layout.width}px`, height: `${layout.height}px`, minWidth: '100%' }">
      <svg class="pointer-events-none absolute inset-0 size-full" :width="layout.width" :height="layout.height">
        <defs>
          <marker id="yd-topo-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
            <path d="M0,0 L8,4 L0,8 Z" class="topo-arrow" />
          </marker>
        </defs>
        <path
          v-for="(p, i) in edgePaths" :key="i" :d="p.d" fill="none" stroke-width="1.5"
          :class="p.cls" marker-end="url(#yd-topo-arrow)"
        >
          <title>{{ p.title }}</title>
        </path>
      </svg>
      <button
        v-for="p in placedList" :key="p.name" type="button"
        class="absolute flex cursor-pointer flex-col justify-center rounded-lg border bg-background px-3 py-1.5 text-left shadow-sm transition-colors hover:border-primary"
        :class="stateCls[p.node.state ?? ''] || stateCls.exited"
        :style="{ left: `${p.x}px`, top: `${p.y}px`, width: `${NODE_W}px`, height: `${NODE_H}px` }"
        :title="$t('components.ydTopology.nodeTip', { name: p.name, image: p.node.image })"
        @click="onNode(p.name)"
      >
        <span class="flex min-w-0 items-center gap-1.5">
          <span class="inline-block size-1.5 shrink-0 rounded-full" :class="dotCls[p.node.state ?? ''] || dotCls.exited" />
          <span class="truncate font-mono text-[13px] font-medium">{{ p.name }}</span>
          <span
            v-if="badgeCls(p.node)"
            class="ml-auto shrink-0 rounded-full px-1.5 py-px text-[10px] leading-4"
            :class="badgeCls(p.node)"
          >{{ badgeText(p.node) }}</span>
          <span v-else class="ml-auto shrink-0 text-[10px] text-muted-foreground">{{ badgeText(p.node) }}</span>
        </span>
        <span class="truncate font-mono text-[11px] text-muted-foreground" :title="p.node.image">{{ p.node.image }}</span>
        <span v-if="p.node.ports?.length" class="truncate font-mono text-[10px] text-muted-foreground/80">{{ p.node.ports.join(' ') }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
/* 主题色走 fa 的 oklch 变量（shadcn 体系），明暗主题自动适配 */
.topo-edge {
  stroke: oklch(var(--muted-foreground));
  opacity: 0.55;
}
.topo-edge-dash {
  stroke: oklch(var(--muted-foreground));
  opacity: 0.45;
  stroke-dasharray: 5 4;
}
.topo-edge-bad {
  stroke: oklch(var(--destructive));
  opacity: 0.8;
}
.topo-arrow {
  fill: oklch(var(--muted-foreground));
  opacity: 0.55;
}
</style>
