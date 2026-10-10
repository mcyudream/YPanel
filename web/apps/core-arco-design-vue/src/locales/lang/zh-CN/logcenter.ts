// logcenter 域词条（B26-full）：日志中心页（src/views/logcenter/index.vue）
export default {
  title: '日志中心',
  description: '集中日志检索（VictoriaLogs / LogsQL）：多节点日志自动汇聚聚合，历史可查、直方图定位、字段切面过滤',

  status: {
    notConnected: '集中日志未接入',
    recheck: '重新检测',
    guide1a: '应用商店安装 ',
    guide1b: '（每台需要汇聚日志的节点各一份，面板自动发现）；',
    guide2a: '同节点安装 ',
    guide2Name: 'Vector 日志采集器',
    guide2b: '（默认参数即可，自动推送本节点 VL）；',
    guide3: '无需任何地址配置——装好后本页自动发现并聚合全部节点的日志。',
    nodes: '聚合节点',
    found: '已发现',
    notInstalled: '未安装',
    offline: '离线',
    aggregating: '聚合中',
    connected: '已接入',
    install: '一键安装',
    installQueued: '已发起安装任务（VictoriaLogs + Vector），进度见任务中心，安装完成后本页自动发现',
    installFail: '一键安装失败',
    noSource: '商店源不可用（请先在应用商店配置并启用源）',
  },

  retention: {
    label: '保留期',
    inputTitle: '如 7d / 30d / 90d / 1y',
    title: '统一设置保留期',
    confirm: '将把所有已接入节点的 VictoriaLogs 保留期设为 {period}，容器会重启数秒（查询短暂中断），过期数据由 VL 自动清理。确认执行？',
    applied: '已应用至 {n} 个节点',
    partialFailed: '部分节点失败',
    setFailed: '设置失败',
  },

  fields: {
    label: '字段',
    searchPlaceholder: '搜索字段或字段值',
    follow: '关注字段',
    unfollow: '取消关注',
    noData: '（无数据）',
  },

  filters: {
    clear: '清空过滤({n})',
    hitToggle: '命中 {n} 条，点击{action}过滤',
    addTo: '加入',
    removeFrom: '移出',
  },

  toolbar: {
    simple: '简易',
    advanced: '高级',
    historyQuick: '历史与常用',
    manage: '管理…',
    to: '至',
    keywordPlaceholder: '关键字…',
    search: '查询',
    matchTitle: '词匹配走倒排索引，大日志量下最快；正则为逐行扫描最慢',
  },

  match: {
    word: '词匹配（快）',
    phrase: '短语包含',
    regex: '正则（慢）',
  },

  time: {
    '15m': '最近 15 分钟',
    '1h': '最近 1 小时',
    '6h': '最近 6 小时',
    '24h': '最近 24 小时',
    '7d': '最近 7 天',
    custom: '自定义范围',
  },

  limit: {
    l100: '上限 100 条',
    l500: '上限 500 条',
    l1000: '上限 1 千条',
    l5000: '上限 5 千条',
    l20000: '上限 2 万条',
  },

  results: {
    histTitle: '点击柱子可缩放到该时段',
    hitsPrefix: '命中',
    hitsSuffix: '条',
    loaded: '已加载 {n} 条',
    pageLimit: '单页已达上限',
    newestFirst: '最新在前',
    oldestFirst: '最旧在前',
    querying: 'VictoriaLogs 查询中…',
    noResults: '无命中结果（提示：采集器只收集启动后的新日志）',
    emptyHint: '配置接通后，按时间范围 / 字段切面 / 关键字检索历史日志',
    copyRaw: '复制原文',
    copiedRaw: '已复制原文',
    copyJson: '复制 JSON',
    copiedJson: '已复制 JSON',
    loadMore: '加载更早（已加载 {loaded}/{total}）',
  },

  history: {
    title: '查询历史与常用查询',
    empty: '暂无记录（执行查询后自动记录）',
    pinTitle: '置顶为常用查询',
    remarkPlaceholder: '命名…',
    use: '使用',
    clearHistory: '清空历史',
    emptyQuery: '(空查询)',
  },

  search: {
    noInstance: '未发现可用的 VictoriaLogs 实例（请先在节点上安装 VictoriaLogs 应用）',
    customStartRequired: '请选择开始时间',
    queryFailed: '查询失败',
    loadFailed: '加载失败',
  },
}
