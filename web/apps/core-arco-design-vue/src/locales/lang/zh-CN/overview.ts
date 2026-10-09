// overview 域词条（B26-full）：主机概览页（src/views/overview/index.vue）
export default {
  title: '主机概览',
  desktop: '桌面工作台',
  desktopTip: '切换到桌面工作台',
  last10m: '近 10 分钟',
  mem: '内存',
  collectFailed: '采集失败',

  sys: {
    title: '系统概览',
    collecting: '采集中…',
    hostname: '主机名',
    os: '操作系统',
    kernel: '内核版本',
    unknownModel: '未知型号',
    cores: '{n} 核',
    swapDisabled: '未启用',
    load: '负载 (1/5/15m)',
    netTotal: '网络总量',
    uptime: '已运行',
    uptimeDHM: '{d} 天 {h} 时 {m} 分',
    uptimeHM: '{h} 时 {m} 分',
    sysTime: '系统时间',
    memPercent: '内存 %',
  },

  net: {
    title: '网络速率',
    down: '下行',
    up: '上行',
  },

  trend: {
    title: 'CPU / 内存',
  },

  docker: {
    title: '用量统计',
    subtitle: 'Docker 资源占用',
    lastUpdate: '数据最后更新：{time}',
    refreshTip: '刷新（system df 实算 size，资源多时较慢）',
    loadFailed: 'Docker 用量采集失败：{msg}',
    collecting: '采集中…（system df 实算 size，资源多时较慢）',
    containers: '容器',
    images: '镜像',
    volumes: '存储卷',
    containersCount: '{n} 个 · 根目录及写入数据',
    imagesCount: '{n} 个 · 包含中间镜像',
    volumesCount: '{n} 个',
    buildCache: '构建缓存',
    buildCacheCount: '{n} 条',
    pruneTitle: '清空构建缓存',
    pruneConfirm: '确认清空全部未使用的构建缓存？该操作不可恢复。',
    pruneButton: '确认清空',
    pruneDone: '已清空构建缓存，释放 {size}',
    pruneFailed: '清空构建缓存失败',
    networks: '网络',
    hostPorts: '主机端口',
    hostPortsTip: '去重后 tcp/udp',
  },

  disk: {
    title: '磁盘用量',
  },

  notify: {
    title: '最近通知',
    viewAll: '查看全部',
  },

  du: {
    title: '磁盘占用分析',
    tip: '点击色块下钻子目录',
    statFailed: '统计失败',
    statFailedMsg: '统计失败：{msg}',
    scanning: '统计中…（du 实算，目录大时较慢）',
    total: '合计 {size}',
    dataTime: '数据时间 {time}（24h 缓存）',
    refreshTip: '重新实算当前目录（绕过缓存）',
  },

  // M44 首页 v2：横幅 / 状态环 / 管理对象 / 到期证书
  banner: {
    night: '夜深了',
    morning: '早上好',
    afternoon: '下午好',
    evening: '晚上好',
  },

  rings: {
    load: '负载',
    disk: '磁盘',
    loadLow: '负载正常',
    loadMid: '负载偏高',
    loadHigh: '负载过高',
    memTotal: '总量',
    memUsed: '已用',
    memAvail: '可用',
  },

  assets: {
    sites: '网站',
    databases: '数据库',
    containers: '容器',
    images: '镜像',
    certs: '证书',
    cron: '计划任务',
    nodes: '节点',
    store: '应用商店',
    enter: '进入',
    running: '运行中',
    expiring: '{n} 张将到期',
    expired: '{n} 张已过期',
    allOk: '全部正常',
    online: '在线',
    dockerDown: 'Docker 不可用',
  },

  certs: {
    title: '到期证书',
    allOk: '30 天内无证书到期',
    expired: '已过期',
    daysLeft: '剩余 {n} 天',
  },
}
