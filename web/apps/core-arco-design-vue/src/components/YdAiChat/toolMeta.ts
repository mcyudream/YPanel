// AI 内置系统工具的展示元数据：流程卡片（YdAiProcess）与系统工具管理页共用。
export interface AiToolMeta {
  label: string
  icon: string
  desc: string
}

export const TOOL_META: Record<string, AiToolMeta> = {
  get_overview: { label: '服务器概览', icon: 'i-lucide:activity', desc: '获取 CPU / 内存 / 磁盘 / 负载 / 网络实时数据' },
  list_containers: { label: '容器列表', icon: 'i-lucide:boxes', desc: '列出全部 Docker 容器（名称/镜像/状态/端口）' },
  container_action: { label: '容器操作', icon: 'i-lucide:play-circle', desc: '对容器执行启动 / 停止 / 重启（破坏性操作前会先确认）' },
  list_sites: { label: '站点列表', icon: 'i-lucide:globe', desc: '列出面板管理的全部网站站点' },
  read_file: { label: '读取文件', icon: 'i-lucide:file-text', desc: '读取服务器上的文本文件（前 100KB）' },
  run_in_workspace: { label: '沙箱执行', icon: 'i-lucide:terminal', desc: '在 AI 工作空间沙箱目录执行 shell 命令 / 脚本' },
  list_database_instances: { label: '数据库实例', icon: 'i-lucide:database', desc: '列出面板管理的全部数据库实例' },
  query_database: { label: '执行查询', icon: 'i-lucide:database-zap', desc: '对数据库执行只读 SQL（仅 SELECT/SHOW/DESC/EXPLAIN，最多 40 行）' },
  save_memory: { label: '保存记忆', icon: 'i-lucide:brain-circle', desc: '把对话中值得长期记住的经验沉淀为长期记忆' },
  read_knowledge: { label: '知识库查询', icon: 'i-lucide:book-search', desc: '按需查询注入的知识文档全文或检索命中章节' },
}

export function toolMetaOf(name: string): AiToolMeta {
  return TOOL_META[name] || { label: name, icon: 'i-lucide:wrench', desc: '' }
}
