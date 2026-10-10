<script setup lang="ts">
// 桌面工作台 v2：YudreamWebOS 全量桌面壳 + YPanel 特异化。
// ① 业务应用多为多实例（高频可开多窗）；容器/应用/站点详情为「隐藏应用」，
//    列表下钻经开窗而非路由跳转（嵌入上下文见 embed.ts，避免顶掉 /desktop 路由）。
// ② 主机概览为 webos 原生卡片形态（OverviewApp，YwCard + --yw token）。
// ③ 桌面右侧栏小组件：库内置时钟/日历 + YPanel 真数据卡（系统/磁盘/网络）。
import type { AppDefinition, MenuItem } from '@yudream/yudream-webos-core'
import {
  YwControlCenter,
  YwDesktop,
  YwDock,
  YwLaunchpad,
  YwMenubar,
  YwNotificationCenter,
  YwQuickLaunch,
  YwWidgetGallery,
} from '@yudream/yudream-webos-arco'
import {
  shortcuts,
  useAppRegistry,
  useMenuBar,
  useSystemSettings,
  useWebOS,
  WebOSProvider,
} from '@yudream/yudream-webos-vue'
import { useAppsStore, useWindowsStore } from '@yudream/yudream-webos-arco'
import { computed, defineComponent, h, markRaw, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { i18n } from '@/locales'
import ContainerAppDetailApp from './apps/ContainerAppDetailApp.vue'
import YdWebBrowser from '@/components/YdWebBrowser/index.vue'
import { appIconSrc, storeApi } from '@/api/modules/store'

import ContainerDetailApp from './apps/ContainerDetailApp.vue'
import FileApp from './apps/FileApp.vue'
import NodeDetailApp from './apps/NodeDetailApp.vue'
import ProcessesApp from './apps/ProcessesApp.vue'
import SiteDetailApp from './apps/SiteDetailApp.vue'
import TerminalApp from './apps/TerminalApp.vue'
import TextEditorApp from './apps/TextEditorApp.vue'
import { YwEmbedKey } from './embed'
import { lastFocusedWin } from './focus-scene'
import { YwSettingsApp } from '@yudream/yudream-webos-arco'
import { wallpapers } from './wallpapers'
import ClockWidget from './widgets/ClockWidget.vue'
import DiskWidget from './widgets/DiskWidget.vue'
import DockerWidget from './widgets/DockerWidget.vue'
import NetWidget from './widgets/NetWidget.vue'
import ShortcutsWidget from './widgets/ShortcutsWidget.vue'
import SystemWidget from './widgets/SystemWidget.vue'

defineOptions({
  name: 'DesktopIndex',
})

const router = useRouter()
const windowsStore = useWindowsStore()
const registry = useAppRegistry()
const { setSystemMenus } = useMenuBar()
const theme = useSystemSettings()
const os = useWebOS()
const appsStore = useAppsStore()

// AI 焦点窗口感知：记录最近聚焦的工作窗口（AI/设置除外），供智能窗对话注入场景上下文
watch(() => windowsStore.focusedId, (id) => {
  const win = id ? windowsStore.windows.find(w => w.id === id) : undefined
  if (!win || win.appId === 'ai' || win.appId === 'settings') {
    return
  }
  lastFocusedWin.value = { windowId: win.id, appId: win.appId, title: win.title, launchOptions: win.launchOptions }
})

const showQuickLaunch = ref(false)
const showLaunchpad = ref(false)
const showControlCenter = ref(false)
const showNotificationCenter = ref(false)
const showWidgetGallery = ref(false)

/** 小组件定义（随概览应用注册；真数据轮询在组件内部） */
const ypanelWidgets: AppDefinition['widgets'] = [
  { id: 'yp-system', name: i18n.global.t('desktop.widgets.yp-system'), sizes: ['medium'], component: markRaw(SystemWidget) },
  { id: 'yp-disk', name: i18n.global.t('desktop.widgets.yp-disk'), sizes: ['small', 'medium'], component: markRaw(DiskWidget) },
  { id: 'yp-net', name: i18n.global.t('desktop.widgets.yp-net'), sizes: ['small'], component: markRaw(NetWidget) },
  { id: 'yp-clock', name: i18n.global.t('desktop.widgets.yp-clock'), sizes: ['small'], component: markRaw(ClockWidget) },
  { id: 'yp-docker', name: i18n.global.t('desktop.widgets.yp-docker'), sizes: ['medium'], component: markRaw(DockerWidget) },
  { id: 'yp-shortcuts', name: i18n.global.t('desktop.widgets.yp-shortcuts'), sizes: ['small', 'medium'], component: markRaw(ShortcutsWidget) },
]

/** 高频应用：上桌面图标 + 启动台；与当前经典面板菜单语义对齐 */
const ypanelApps: AppDefinition[] = [
  { id: 'overview', name: i18n.global.t('desktop.apps.overview'), icon: 'i-lucide-gauge', component: () => import('./apps/OverviewApp.vue'), category: 'system', keywords: 'gailan overview host zhuji jiankong', defaultSize: { width: 900, height: 620 }, launchpad: { order: 10 }, widgets: ypanelWidgets },
  { id: 'file', name: i18n.global.t('desktop.apps.file'), icon: 'i-lucide-folder-open', component: FileApp, category: 'system', keywords: 'wenjian file wjg', defaultSize: { width: 960, height: 620 }, singleton: false, multiInstance: true, launchpad: { order: 190 } },
  { id: 'terminal', name: i18n.global.t('desktop.apps.terminal'), icon: 'i-lucide-square-terminal', component: TerminalApp, category: 'develop', keywords: 'zhongduan terminal shell ssh', defaultSize: { width: 880, height: 560 }, singleton: false, multiInstance: true, launchpad: { order: 200 } },
  { id: 'container', name: i18n.global.t('desktop.apps.container'), icon: 'i-lucide-container', component: () => import('@/views/container/tabs/AppsTab.vue'), category: 'system', keywords: 'rongqi container docker bianpai compose', defaultSize: { width: 960, height: 620 }, singleton: false, multiInstance: true, launchpad: { order: 60 } },
  { id: 'database', name: i18n.global.t('desktop.apps.database'), icon: 'i-lucide-database', component: () => import('@/views/database/index.vue'), category: 'database', keywords: 'shujuku database mysql redis', defaultSize: { width: 960, height: 620 }, singleton: false, multiInstance: true, launchpad: { order: 70 } },
  { id: 'sites', name: i18n.global.t('desktop.apps.sites'), icon: 'i-lucide-globe', component: () => import('@/views/sites/index.vue'), category: 'system', keywords: 'wangzhan site nginx', defaultSize: { width: 960, height: 600 }, singleton: false, multiInstance: true, launchpad: { order: 20 } },
  { id: 'store', name: i18n.global.t('desktop.apps.store'), icon: 'i-lucide-shopping-bag', component: () => import('@/views/store/index.vue'), category: 'tool', keywords: 'yingyong shangdian store app market', defaultSize: { width: 960, height: 620 }, launchpad: { order: 80 } },
  { id: 'ai', name: i18n.global.t('desktop.apps.ai'), icon: 'i-lucide-sparkles', component: () => import('@/views/ai/chat.vue'), category: 'ai', keywords: 'zhineng ai chat llm gpt', defaultSize: { width: 920, height: 620 }, launchpad: { order: 30 } },
  { id: 'processes', name: i18n.global.t('desktop.apps.processes'), icon: 'i-lucide-activity', component: ProcessesApp, category: 'system', keywords: 'jincheng process fuwu service systemd', defaultSize: { width: 960, height: 600 }, launchpad: { order: 210 } },
  { id: 'monitor', name: i18n.global.t('desktop.apps.monitor'), icon: 'i-lucide-chart-line', component: () => import('@/views/manage/monitor.vue'), category: 'monitor', keywords: 'lishi jiankong monitor history trend qushi', defaultSize: { width: 960, height: 600 }, launchpad: { order: 220 } },
]

/**
 * 全量应用：与经典面板菜单对齐，桌面高频之外的低频应用不播种桌面（desktop.show=false），
 * 经启动台 / Ctrl+K 搜索 / Dock 运行中可达。路径语义与 router/modules 一一对应。
 */
const launcherApps: AppDefinition[] = [
  { id: 'certs', name: i18n.global.t('desktop.apps.certs'), icon: 'i-lucide-shield-check', component: () => import('@/views/certs/index.vue'), category: 'system', keywords: 'zhengshu cert tls https acme', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 30 } },
  { id: 'runtimes', name: i18n.global.t('desktop.apps.runtimes'), icon: 'i-lucide-boxes', component: () => import('@/views/runtimes/index.vue'), category: 'system', keywords: 'yunxing php node python java go runtime', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 40 } },
  { id: 'container-list', name: i18n.global.t('desktop.apps.container-list'), icon: 'i-lucide-box', component: () => import('@/views/container/tabs/ListTab.vue'), category: 'system', keywords: 'rongqi container list liebiao', defaultSize: { width: 960, height: 620 }, singleton: false, multiInstance: true, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 50 } },
  { id: 'container-images', name: i18n.global.t('desktop.apps.container-images'), icon: 'i-lucide-disc-3', component: () => import('@/views/container/tabs/ImagesTab.vue'), category: 'system', keywords: 'jingxiang image docker mirror', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 55 } },
  { id: 'docker-networks', name: i18n.global.t('desktop.apps.docker-networks'), icon: 'i-lucide-network', component: () => import('@/views/container/tabs/NetworksTab.vue'), category: 'network', keywords: 'wangluo network docker bridge', defaultSize: { width: 920, height: 580 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 65 } },
  { id: 'docker-volumes', name: i18n.global.t('desktop.apps.docker-volumes'), icon: 'i-lucide-archive', component: () => import('@/views/container/tabs/VolumesTab.vue'), category: 'system', keywords: 'cunchujuan volume juan', defaultSize: { width: 920, height: 580 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 75 } },
  { id: 'docker-settings', name: i18n.global.t('desktop.apps.docker-settings'), icon: 'i-lucide-settings-2', component: () => import('@/views/container/tabs/SettingsTab.vue'), category: 'system', keywords: 'docker peizhi config daemon', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 85 } },
  { id: 'cron', name: i18n.global.t('desktop.apps.cron'), icon: 'i-lucide-calendar-clock', component: () => import('@/views/cron/index.vue'), category: 'tool', keywords: 'jihua cron task dingshi', defaultSize: { width: 920, height: 580 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 90 } },
  { id: 'dns', name: i18n.global.t('desktop.apps.dns'), icon: 'i-lucide-radar', component: () => import('@/views/dns/index.vue'), category: 'network', keywords: 'neiwang dns jiexi domain', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 95 } },
  { id: 'alert', name: i18n.global.t('desktop.apps.alert'), icon: 'i-lucide-bell-ring', component: () => import('@/views/alert/index.vue'), category: 'monitor', keywords: 'gaojing alert tongzhi notify', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 230 } },
  { id: 'firewall', name: i18n.global.t('desktop.apps.firewall'), icon: 'i-lucide-flame', component: () => import('@/views/firewall/index.vue'), category: 'network', keywords: 'fanghuoqiang firewall ufw port', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 240 } },
  { id: 'nat', name: i18n.global.t('desktop.apps.nat'), icon: 'i-lucide-arrow-left-right', component: () => import('@/views/nat/index.vue'), category: 'network', keywords: 'nat zhuanfa forward port', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 250 } },
  { id: 'hosts', name: i18n.global.t('desktop.apps.hosts'), icon: 'i-lucide-list-tree', component: () => import('@/views/hosts/index.vue'), category: 'network', keywords: 'hosts jizuoming fenfa', defaultSize: { width: 920, height: 580 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 260 } },
  { id: 'nodes', name: i18n.global.t('desktop.apps.nodes'), icon: 'i-lucide-server', component: () => import('@/views/nodes/index.vue'), category: 'system', keywords: 'jiedian node server duo', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 270 } },
  { id: 'selfupdate', name: i18n.global.t('desktop.apps.selfupdate'), icon: 'i-lucide-settings', component: () => import('@/views/selfupdate/index.vue'), category: 'system', keywords: 'mianban shezhi settings update gengxin', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 280 } },
  { id: 'vpn', name: i18n.global.t('desktop.apps.vpn'), icon: 'i-lucide-share-2', component: () => import('@/views/vpn/index.vue'), category: 'network', keywords: 'zuwang vpn easytier mesh', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 96 } },
  { id: 'browser', name: i18n.global.t('desktop.apps.browser'), icon: 'i-lucide-globe', component: () => import('./apps/BrowserApp.vue'), category: 'tool', keywords: 'liulanqi browser neiwang proxy daili fangwen 内网 浏览器 代理', defaultSize: { width: 960, height: 640 }, singleton: false, multiInstance: true, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 97 } },
  // ── 全量纳管补齐（2026-10-09）：数据库管理台/监控域/智能域其余页面，路径语义与 router/modules 一一对应 ──
  { id: 'dbadmin', name: i18n.global.t('desktop.apps.dbadmin'), icon: 'i-lucide-database-zap', component: () => import('@/views/dbadmin/index.vue'), category: 'database', keywords: 'dbadmin shujuku guanlitai database admin gongzuotai', defaultSize: { width: 1080, height: 680 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 72 } },
  { id: 'probe', name: i18n.global.t('desktop.apps.probe'), icon: 'i-lucide-radar', component: () => import('@/views/probe/index.vue'), category: 'monitor', keywords: 'tanzhen probe jiankonghttp tcp', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 226 } },
  { id: 'logcenter', name: i18n.global.t('desktop.apps.logcenter'), icon: 'i-lucide-scroll-text', component: () => import('@/views/logcenter/index.vue'), category: 'monitor', keywords: 'rizhizhongxin logcenter log journal rizhi', defaultSize: { width: 1080, height: 640 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 236 } },
  { id: 'ai-providers', name: i18n.global.t('desktop.apps.ai-providers'), icon: 'i-lucide-plug', component: () => import('@/views/ai/providers.vue'), category: 'ai', keywords: 'gongyingshang provider moxing model llm api', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 31 } },
  { id: 'ai-knowledge', name: i18n.global.t('desktop.apps.ai-knowledge'), icon: 'i-lucide-book-open', component: () => import('@/views/ai/knowledge.vue'), category: 'ai', keywords: 'zhishiku knowledge wendang rag', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 32 } },
  { id: 'ai-workspace', name: i18n.global.t('desktop.apps.ai-workspace'), icon: 'i-lucide-folder-code', component: () => import('@/views/ai/workspace.vue'), category: 'ai', keywords: 'gongzuoqu workspace daima code', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 33 } },
  { id: 'ai-memory', name: i18n.global.t('desktop.apps.ai-memory'), icon: 'i-lucide-brain', component: () => import('@/views/ai/memory.vue'), category: 'ai', keywords: 'jiyi memory huiyi', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 34 } },
  { id: 'ai-tools', name: i18n.global.t('desktop.apps.ai-tools'), icon: 'i-lucide-wrench', component: () => import('@/views/ai/tools.vue'), category: 'ai', keywords: 'gongju tools tool zhineng', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 35 } },
  { id: 'ai-skills', name: i18n.global.t('desktop.apps.ai-skills'), icon: 'i-lucide-puzzle', component: () => import('@/views/ai/skills.vue'), category: 'ai', keywords: 'jineng skill skillbao jishubao', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 36 } },
  { id: 'ai-mcp', name: i18n.global.t('desktop.apps.ai-mcp'), icon: 'i-lucide-plug-zap', component: () => import('@/views/ai/mcp.vue'), category: 'ai', keywords: 'mcp server fuwu xieyi', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 37 } },
]

/**
 * 管理域六页（用户/审计/安全/备份/系统/存储）：admin 专属，
 * 与经典菜单 auth: ['admin'] 一致——非 admin 账号不注册（启动台/搜索不可见）。
 */
const manageApps: AppDefinition[] = [
  { id: 'manage-user', name: i18n.global.t('desktop.apps.manage-user'), icon: 'i-lucide-users', component: () => import('@/views/manage/user.vue'), category: 'system', keywords: 'yonghu user zhanghu account quanxian', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 290 } },
  { id: 'manage-audit', name: i18n.global.t('desktop.apps.manage-audit'), icon: 'i-lucide-scroll', component: () => import('@/views/manage/audit.vue'), category: 'system', keywords: 'shenji audit caozuo rizhi', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 291 } },
  { id: 'manage-security', name: i18n.global.t('desktop.apps.manage-security'), icon: 'i-lucide-shield', component: () => import('@/views/manage/security.vue'), category: 'system', keywords: 'anquan security fanghu shezhi', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 292 } },
  { id: 'manage-backups', name: i18n.global.t('desktop.apps.manage-backups'), icon: 'i-lucide-save', component: () => import('@/views/manage/backups.vue'), category: 'system', keywords: 'beifen backup huanyuan rescue', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 293 } },
  { id: 'manage-system', name: i18n.global.t('desktop.apps.manage-system'), icon: 'i-lucide-wrench', component: () => import('@/views/manage/system.vue'), category: 'system', keywords: 'xitong system guanli weihu weibu', defaultSize: { width: 920, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 294 } },
  { id: 'manage-storage', name: i18n.global.t('desktop.apps.manage-storage'), icon: 'i-lucide-cloud-upload', component: () => import('@/views/manage/storage.vue'), category: 'system', keywords: 'cunchu storage yun duixiang', defaultSize: { width: 960, height: 600 }, dock: { showInDock: false }, desktop: { show: false }, launchpad: { order: 295 } },
]

/** 管理域注册开关：与经典菜单权限同口径（未启用账号体系时视为有权限） */
const isAdminAccount = useAppAuth().auth(['admin'])

/** 系统设置（webos 原生：壁纸/强调色/深浅/关于）：仅右键「系统设置…」入口可达 */
const settingsApp: AppDefinition = {
  id: 'settings',
  name: i18n.global.t('desktop.apps.settings'),
  icon: 'i-lucide-settings',
  component: markRaw(YwSettingsApp),
  category: 'system',
  dock: { showInDock: false },
  desktop: { show: false },
}

/** 桌面文本文件编辑器（隐藏应用）：桌面文件图标双击时经 openApp('text-editor') 承载 */
const textEditorApp: AppDefinition = {
  id: 'text-editor',
  name: i18n.global.t('desktop.apps.textEditor'),
  icon: 'i-lucide-file-text',
  component: markRaw(TextEditorApp),
  category: 'tool',
  singleton: false,
  multiInstance: true,
  dock: { showInDock: false },
  desktop: { show: false },
  launchpad: { show: false },
}

/** 详情承载：隐藏应用（不进 Dock/启动台/桌面播种），multiInstance 每次开新窗、标题带实例名 */
const detailApps: AppDefinition[] = [
  { id: 'container-detail', name: i18n.global.t('desktop.apps.container-detail'), icon: 'i-lucide-container', component: ContainerDetailApp, singleton: false, multiInstance: true, defaultSize: { width: 960, height: 640 }, dock: { showInDock: false }, launchpad: { show: false } },
  { id: 'container-app-detail', name: i18n.global.t('desktop.apps.container-app-detail'), icon: 'i-lucide-package', component: ContainerAppDetailApp, singleton: false, multiInstance: true, defaultSize: { width: 960, height: 640 }, dock: { showInDock: false }, launchpad: { show: false } },
  { id: 'site-detail', name: i18n.global.t('desktop.apps.site-detail'), icon: 'i-lucide-globe', component: SiteDetailApp, singleton: false, multiInstance: true, defaultSize: { width: 960, height: 640 }, dock: { showInDock: false }, launchpad: { show: false } },
  { id: 'nodes-detail', name: i18n.global.t('desktop.apps.nodes-detail'), icon: 'i-lucide-server', component: NodeDetailApp, singleton: false, multiInstance: true, defaultSize: { width: 960, height: 640 }, dock: { showInDock: false }, launchpad: { show: false } },
]

/** Dock 固定：高频 8 个（其余应用运行中自动出现） */
const pinnedDockIds = ['overview', 'file', 'terminal', 'container', 'database', 'sites', 'store', 'ai']

/** Alt+数字 一键打开/聚焦的高频应用序列（与 Dock 固定一致） */
const launchSlots = ['overview', 'file', 'terminal', 'container', 'database', 'sites', 'store', 'ai']

/** 内置壁纸（随资产打包）+ 默认渐变兜底；壁纸选择在控制中心，随系统设置持久化 */
const defaultWallpaper = { src: 'linear-gradient(160deg, #0f172a 0%, #1e293b 45%, #334155 100%)' }
const currentWallpaper = computed(() => theme.settings.wallpaper ?? defaultWallpaper)

// 置空 menubar 的默认菜单组。注意该 prop 不能在模板里写字面量空数组——
// UnoCSS 会把源码里的字面量提取为候选类名并生成非法 CSS（Unclosed bracket）
const noFallbackMenus: string[] = []

const systemMenus: MenuItem[] = [
  {
    id: 'system',
    label: i18n.global.t('desktop.menu.system'),
    submenu: [
      { id: 'back-classic', label: i18n.global.t('desktop.menu.backClassic'), action: () => backToClassic() },
      { id: 'export-layout', label: i18n.global.t('desktop.menu.syncLayoutNow'), action: () => syncLayoutNow() },
      { id: 'import-layout', label: i18n.global.t('desktop.importLayout'), action: () => importLayout() },
    ],
  },
]

/** 弹窗容器注入：窗口内业务弹窗（FaModal）约束在所属窗口内（遮罩只盖本窗） */
provide('fa:modal-container', computed(() => {
  const id = windowsStore.focusedId
  if (!id) {
    return undefined
  }
  return document.querySelector(`[data-window-id="${id}"] .yw-window-body`)
}))

/** 嵌入上下文：窗口内页面用它开新窗（替代 router.push，防止顶掉 /desktop 路由） */
provide(YwEmbedKey, {
  openApp: (appId, opts) => {
    os.wm.open(appId, { title: opts?.title, launchOptions: opts?.launchOptions })
  },
  closeWindow: (windowId) => {
    os.wm.close(windowId)
  },
})

/** 拖文件到 Dock 图标：终端=新开终端窗并填入路径；文件管理=新窗打开所在目录（多文件取首个所在目录） */
function onDockFileDrop({ appId, dataTransfer }: { appId: string, dataTransfer: DataTransfer }) {
  const raw = dataTransfer.getData('application/x-ypanel-files')
  if (!raw) {
    return
  }
  try {
    const payload = JSON.parse(raw) as { node: string, items: Array<{ path: string, name: string, isDir: boolean }> }
    const paths = payload.items.map(x => x.path)
    if (appId === 'terminal') {
      os.wm.open('terminal', { title: i18n.global.t('desktop.apps.terminal'), launchOptions: { dropPaths: paths } })
    }
    else if (appId === 'file' && paths.length) {
      const dir = paths[0].slice(0, paths[0].lastIndexOf('/')) || '/'
      os.wm.open('file', { title: dir, launchOptions: { dir, node: payload.node } })
    }
  }
  catch {}
}

for (const app of [...ypanelApps, ...launcherApps, ...(isAdminAccount ? manageApps : []), ...detailApps, settingsApp, textEditorApp]) {
  registry.register(app)
}

// P3 商店应用桌面注册：已安装且声明 adminUI 的管理工具 → 桌面/启动台图标，点击经内网浏览器（gw 会话式反代）打开管理界面
const storeDesktopApps = new Map<string, { url: string }>()
async function refreshStoreDesktopApps() {
  try {
    const infos = await storeApi.installed()
    const wanted = new Map<string, { name: string, icon: string, url: string }>()
    for (const it of infos) {
      if (!it.adminUI?.url) continue
      wanted.set(`store-${it.composeProject}`, {
        name: it.adminUI.name || it.appName || it.name,
        icon: appIconSrc(it.iconUrl),
        url: it.adminUI.url,
      })
    }
    for (const [id, def] of wanted) {
      if (storeDesktopApps.has(id)) continue
      const url = def.url
      appsStore.register({
        id,
        name: def.name,
        icon: def.icon,
        component: defineComponent({
          name: 'StoreWebUIApp',
          setup() {
            return () => h(YdWebBrowser, { url })
          },
        }),
        singleton: true,
        defaultSize: { width: 1100, height: 720 },
      })
      storeDesktopApps.set(id, { url })
    }
    // 卸载/声明移除：注销应用并清理桌面残留图标
    for (const id of [...storeDesktopApps.keys()]) {
      if (!wanted.has(id)) {
        appsStore.unregister(id)
        storeDesktopApps.delete(id)
        for (const item of os.desktop.list()) {
          if (item.type === 'app' && item.refId === id) {
            os.desktop.remove(item.id)
          }
        }
      }
    }
  }
  catch (e) {
    console.warn('商店应用桌面注册失败', e)
  }
}
void refreshStoreDesktopApps()
/** M44：桌面布局导出/导入（服务器侧备份，跨浏览器/重装恢复） */
const settingsApiLayout = () => import('@/api/modules/settings')

/** M45：布局自动持久化——启动恢复 + 变更防抖上传 */
let lastLayoutSent = ''
let layoutTimer: ReturnType<typeof setTimeout> | null = null

function watchLayoutAutoSync() {
  // 启动恢复：服务器有备份且本地无（新浏览器/清过缓存）
  void (async () => {
    try {
      const { default: settingsApi } = await import('@/api/modules/settings')
      const settings = await settingsApi.get()
      const remote = settings['desktop.layout']
      const local = localStorage.getItem('ypanel.webos.desktop.layout')
      if (remote && !local) {
        localStorage.setItem('ypanel.webos.desktop.layout', remote)
        window.location.reload()
      }
      lastLayoutSent = local || ''
    }
    catch {}
  })()
  // 变更检测：800ms 轮询比对（webos 写 localStorage 无同页事件），防抖 3s 上传
  setInterval(() => {
    const cur = localStorage.getItem('ypanel.webos.desktop.layout') || ''
    if (cur !== lastLayoutSent) {
      if (layoutTimer) clearTimeout(layoutTimer)
      layoutTimer = setTimeout(() => {
        const val = localStorage.getItem('ypanel.webos.desktop.layout') || ''
        if (val === lastLayoutSent) return
        lastLayoutSent = val
        void (async () => {
          try {
            const { default: settingsApi } = await import('@/api/modules/settings')
            await settingsApi.put({ 'desktop.layout': val })
          }
          catch {}
        })()
      }, 3000)
    }
  }, 800)
}

async function syncLayoutNow() {
  const raw = localStorage.getItem('ypanel.webos.desktop.layout')
  const { default: settingsApi } = await settingsApiLayout()
  await settingsApi.put({ 'desktop.layout': raw || '' })
  lastLayoutSent = raw || ''
  os.ui ? os.ui.message('success', i18n.global.t('desktop.layoutSynced')) : useFaToast().success(i18n.global.t('desktop.layoutSynced'))
}

async function importLayout() {
  const { default: settingsApi } = await settingsApiLayout()
  const settings = await settingsApi.get()
  const raw = settings['desktop.layout']
  if (!raw) {
    os.ui ? os.ui.message('info', i18n.global.t('desktop.noRemoteLayout')) : useFaToast().info(i18n.global.t('desktop.noRemoteLayoutShort'))
    return
  }
  localStorage.setItem('ypanel.webos.desktop.layout', raw)
  os.ui ? os.ui.message('success', i18n.global.t('desktop.layoutRestored')) : useFaToast().success(i18n.global.t('desktop.layoutRestored'))
  setTimeout(() => window.location.reload(), 800)
}

// Dock 固定：精选高频 8 个（运行中的应用会自动出现在 Dock）
for (const id of pinnedDockIds) {
  os.dock.pin(id)
}
setSystemMenus(systemMenus)

function backToClassic() {
  localStorage.setItem('ypanel.mode', 'classic')
  router.push('/')
}

onMounted(() => {
  // 恢复持久化的主题/强调色/壁纸（scope: system）
  void theme.load()

  // 桌面右键「编辑小组件」→ 打开画廊（添加/管理；移除与调尺寸在小组件右键菜单）
  window.addEventListener('webos:widgets:edit', () => {
    showWidgetGallery.value = true
  })

  // M45：布局自动持久化（启动恢复 + 变更防抖上传）
  watchLayoutAutoSync()

  // 小组件默认播种（桌面网格，unit=84+gap8；medium 4x2、small 2x2）：仅在没有任何已恢复布局时
  if (os.widgets.listInstances().length === 0) {
    os.widgets.add('yp-system', 'medium', { col: 0, row: 0 })
    os.widgets.add('yp-docker', 'medium', { col: 0, row: 2 })
    os.widgets.add('yp-clock', 'small', { col: 4, row: 0 })
    os.widgets.add('yp-disk', 'small', { col: 4, row: 2 })
    os.widgets.add('yp-net', 'small', { col: 6, row: 0 })
    os.widgets.add('yp-shortcuts', 'small', { col: 6, row: 2 })
  }

  shortcuts.register('Cmd/Ctrl+K', () => {
    showQuickLaunch.value = !showQuickLaunch.value
  })
  shortcuts.register('F4', () => {
    showLaunchpad.value = !showLaunchpad.value
  })

  // 窗口吸附快捷键（作用于聚焦窗口）：←→ 左右半屏、↑ 最大化、↓ 还原/最小化、F 一键平铺
  const snapFocused = (zone: 'left' | 'right' | 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right') => {
    const w = os.wm.focusedWindow()
    if (w) {
      os.wm.snap(w.id, zone)
    }
  }
  shortcuts.register('Ctrl+Alt+ArrowLeft', () => snapFocused('left'))
  shortcuts.register('Ctrl+Alt+ArrowRight', () => snapFocused('right'))
  shortcuts.register('Ctrl+Alt+ArrowUp', () => {
    const w = os.wm.focusedWindow()
    if (w) {
      os.wm.maximize(w.id)
    }
  })
  shortcuts.register('Ctrl+Alt+ArrowDown', () => {
    const w = os.wm.focusedWindow()
    if (!w) {
      return
    }
    if (w.state === 'maximized') {
      os.wm.restore(w.id)
    }
    else {
      os.wm.minimize(w.id)
    }
  })
  shortcuts.register('Ctrl+Alt+F', () => {
    os.wm.tileAll()
  })

  // 一键打开/聚焦高频应用：Alt+数字（顺序与 Dock 固定一致），输入态同样生效。
  // 不让路给 xterm：真键盘下 xterm 对已消化的键会 stopPropagation，冒泡监听自然收不到；
  // 这里只在事件到达时切换，不抢终端自己的按键。
  launchSlots.forEach((appId, idx) => {
    shortcuts.register(`Alt+${idx + 1}`, (e: KeyboardEvent) => {
      // 聚焦该应用最近的窗口（最小化的经 core focus 自动还原）；无窗则新开
      const wins = os.wm.windowsOfApp(appId)
      if (wins.length) {
        os.wm.focus(wins[wins.length - 1].id)
      }
      else {
        os.openApp(appId)
      }
      e.preventDefault()
    })
  })
})

onBeforeUnmount(() => {
  shortcuts.unregister('Cmd/Ctrl+K')
  shortcuts.unregister('F4')
  shortcuts.unregister('Ctrl+Alt+ArrowLeft')
  shortcuts.unregister('Ctrl+Alt+ArrowRight')
  shortcuts.unregister('Ctrl+Alt+ArrowUp')
  shortcuts.unregister('Ctrl+Alt+ArrowDown')
  shortcuts.unregister('Ctrl+Alt+F')
  launchSlots.forEach((_, idx) => {
    shortcuts.unregister(`Alt+${idx + 1}`)
  })
})
</script>

<template>
  <WebOSProvider :wallpaper="currentWallpaper">
    <YwMenubar title="YPanel" :show-clock="true" :fallback-menus="noFallbackMenus" @logoclick="showLaunchpad = true">
      <template #tray>
        <span class="yw-tray-action" :title="$t('desktop.menu.backClassic')" @click="backToClassic">
          <i class="i-lucide-panel-left" />
          <span>{{ $t('desktop.tray.classicPanel') }}</span>
        </span>
        <i class="yw-tray-icon i-lucide-search" :title="$t('desktop.tray.focusSearch')" @click="showQuickLaunch = true" />
        <i class="yw-tray-icon i-lucide-layout-grid" :title="$t('desktop.tray.controlCenter')" @click="showControlCenter = !showControlCenter" />
      </template>
    </YwMenubar>

    <YwDesktop :wallpaper="currentWallpaper" />

    <YwDock :drop-apps="['terminal', 'file']" @show-quicklaunch="showQuickLaunch = true" @file-drop="onDockFileDrop" />

    <YwQuickLaunch v-if="showQuickLaunch" @close="showQuickLaunch = false" />
    <YwLaunchpad v-if="showLaunchpad" @close="showLaunchpad = false" />
    <YwControlCenter
      v-if="showControlCenter"
      :wallpapers="wallpapers"
      :current-wallpaper="theme.settings.wallpaper?.src ?? null"
      @close="showControlCenter = false"
    />
    <YwNotificationCenter v-if="showNotificationCenter" @close="showNotificationCenter = false" />
    <YwWidgetGallery v-if="showWidgetGallery" @close="showWidgetGallery = false" />
  </WebOSProvider>
</template>

<style scoped>
.yw-tray-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  margin-right: 4px;
  font-size: 12px;
  color: var(--yw-label-secondary);
  cursor: pointer;
  border-radius: 6px;
  transition: background var(--yw-dur-ui, 0.15s);
}

.yw-tray-action:hover {
  background: rgb(255 255 255 / 12%);
  color: var(--yw-label);
}

.yw-tray-icon {
  padding: 4px;
  font-size: 14px;
  color: var(--yw-label);
  cursor: pointer;
  border-radius: 6px;
  transition: background var(--yw-dur-ui, 0.15s);
}

.yw-tray-icon:hover {
  background: rgb(255 255 255 / 12%);
}
</style>

<!-- 非 scoped：窗口内容是运行时挂载的经典页面组件，不带本组件 scope id。
     仅作用于 webos 壳内（.yw-provider 随桌面路由卸载），经典模式零影响：
     页面页头（FaPageHeader/yp-page-header）在窗口内冗余——窗口标题栏已承载标题与身份 -->
<style>
/* ── webos 嵌入态页面适配（非 scoped：窗口内容为运行时挂载的经典页面组件）──
   仅作用于 .yw-provider（随桌面路由卸载），经典模式零影响。
   ① 页头只留操作条：标题/描述隐藏（窗口标题栏已承载身份），操作按钮保留成紧凑工具条 */
.yw-provider .yp-page-header-main {
  display: none !important;
}

.yw-provider .yp-page-header {
  gap: 8px !important;
  margin: 8px 8px 0 !important;
  padding: 6px 10px !important;
  background: transparent !important;
  border-bottom: none !important;
  border-radius: 8px 8px 0 0;
}

/* ② 内容容器：外边距压半、去硬边框感 */
.yw-provider .yp-page-main {
  margin: 8px !important;
}

.yw-provider .yp-page-main > div {
  border-radius: 8px;
}

/* ③ 窗口内容撑满 */
.yw-provider .yw-window-body > * {
  min-height: 100%;
}
</style>
