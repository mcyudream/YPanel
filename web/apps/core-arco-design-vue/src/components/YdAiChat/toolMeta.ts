// AI 内置系统工具的展示元数据：流程卡片（YdAiProcess）与系统工具管理页共用。
// 展示文案走 i18n（ai.tools.tool.* / ai.tools.risk.*），label/desc 在访问时即时求值（切语言即生效）。
import { i18n } from '@/locales'

export interface AiToolMeta {
  label: string
  icon: string
  desc: string
}

const TOOL_ICONS: Record<string, string> = {
  get_overview: 'i-lucide:activity',
  list_containers: 'i-lucide:boxes',
  container_action: 'i-lucide:play-circle',
  list_sites: 'i-lucide:globe',
  read_file: 'i-lucide:file-text',
  run_in_workspace: 'i-lucide:terminal',
  list_database_instances: 'i-lucide:database',
  query_database: 'i-lucide:database-zap',
  save_memory: 'i-lucide:brain-circle',
  read_knowledge: 'i-lucide:book-search',
}

export function toolMetaOf(name: string): AiToolMeta {
  const icon = TOOL_ICONS[name]
  if (!icon) {
    return { label: name, icon: 'i-lucide:wrench', desc: '' }
  }
  return {
    get label() {
      return i18n.global.t(`ai.tools.tool.${name}.label`)
    },
    icon,
    get desc() {
      return i18n.global.t(`ai.tools.tool.${name}.desc`)
    },
  }
}

/** 风险分级徽标元数据（M31：read 查询自动执行 / write 写入 / danger 危险，均与后端 aiToolDef.Risk 对齐） */
export const RISK_META: Record<string, { label: string, badgeClass: string }> = {
  get read() {
    return { label: i18n.global.t('ai.tools.risk.read'), badgeClass: 'bg-sky-500/10 text-sky-600' }
  },
  get write() {
    return { label: i18n.global.t('ai.tools.risk.write'), badgeClass: 'bg-amber-500/10 text-amber-600' }
  },
  get danger() {
    return { label: i18n.global.t('ai.tools.risk.danger'), badgeClass: 'bg-red-500/10 text-red-500' }
  },
}
