// overview domain messages (B26-full): host overview page (src/views/overview/index.vue)
export default {
  title: 'Host Overview',
  desktop: 'Desktop Workbench',
  desktopTip: 'Switch to Desktop Workbench',
  last10m: 'Last 10 minutes',
  mem: 'Memory',
  collectFailed: 'Failed to collect',

  sys: {
    title: 'System Overview',
    collecting: 'Collecting…',
    hostname: 'Hostname',
    os: 'Operating System',
    kernel: 'Kernel Version',
    unknownModel: 'Unknown model',
    cores: '{n} cores',
    swapDisabled: 'Disabled',
    load: 'Load (1/5/15m)',
    netTotal: 'Network Total',
    uptime: 'Uptime',
    uptimeDHM: '{d}d {h}h {m}m',
    uptimeHM: '{h}h {m}m',
    sysTime: 'System Time',
    memPercent: 'Mem %',
  },

  net: {
    title: 'Network Speed',
    down: 'Down',
    up: 'Up',
  },

  trend: {
    title: 'CPU / Memory',
  },

  docker: {
    title: 'Resource Usage',
    subtitle: 'Docker resource usage',
    lastUpdate: 'Last updated: {time}',
    refreshTip: 'Refresh (system df computes real sizes; slow with many resources)',
    loadFailed: 'Failed to collect Docker usage: {msg}',
    collecting: 'Collecting… (system df computes real sizes; slow with many resources)',
    containers: 'Containers',
    images: 'Images',
    volumes: 'Volumes',
    containersCount: '{n} · root fs & writable data',
    imagesCount: '{n} · incl. intermediate images',
    volumesCount: '{n}',
    buildCache: 'Build Cache',
    buildCacheCount: '{n} entries',
    pruneTitle: 'Clear Build Cache',
    pruneConfirm: 'Clear all unused build cache? This action cannot be undone.',
    pruneButton: 'Clear',
    pruneDone: 'Build cache cleared, {size} freed',
    pruneFailed: 'Failed to clear build cache',
    networks: 'Networks',
    hostPorts: 'Host Ports',
    hostPortsTip: 'Deduplicated tcp/udp',
  },

  disk: {
    title: 'Disk Usage',
  },

  notify: {
    title: 'Recent Notifications',
    viewAll: 'View All',
  },

  du: {
    title: 'Disk Usage Analyzer',
    tip: 'Click a block to drill down into subdirectories',
    statFailed: 'Scan failed',
    statFailedMsg: 'Scan failed: {msg}',
    scanning: 'Scanning… (du computes real sizes; slow for large directories)',
    total: 'Total {size}',
    dataTime: 'Data time {time} (24h cache)',
    refreshTip: 'Re-scan current directory (bypasses cache)',
  },

  // M44 Overview v2: banner / status rings / managed assets / expiring certs
  banner: {
    night: 'Good night',
    morning: 'Good morning',
    afternoon: 'Good afternoon',
    evening: 'Good evening',
  },

  rings: {
    load: 'Load',
    disk: 'Disk',
    loadLow: 'Load is normal',
    loadMid: 'Load is high',
    loadHigh: 'Load is too high',
    memTotal: 'Total',
    memUsed: 'Used',
    memAvail: 'Available',
  },

  assets: {
    sites: 'Sites',
    databases: 'Databases',
    containers: 'Containers',
    images: 'Images',
    certs: 'Certificates',
    cron: 'Cron Tasks',
    nodes: 'Nodes',
    store: 'App Store',
    enter: 'Open',
    running: 'Running',
    expiring: '{n} expiring',
    expired: '{n} expired',
    allOk: 'All healthy',
    online: 'online',
    dockerDown: 'Docker unavailable',
  },

  certs: {
    title: 'Expiring Certificates',
    allOk: 'No certificates expiring within 30 days',
    expired: 'Expired',
    daysLeft: '{n} days left',
  },
}
