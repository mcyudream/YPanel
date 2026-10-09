// selfupdate 域词条（面板设置 / 更新通道）
export default {
  page: {
    title: '面板设置',
    desc: '更新通道：将新版本二进制放入服务器 {dir}（ypanel 前缀命名），此处一键应用；更新前自动备份 ypanel.bak',
    security: '安全设置',
    currentVersion: '当前版本：',
  },
  table: {
    file: '更新文件',
    time: '放入时间',
    empty: '更新通道为空：将新版本二进制 scp 到服务器 {dir} 目录后刷新',
    applying: '更新中…',
    applyRestart: '应用并重启',
  },
  modal: {
    applyTitle: '应用更新',
    applyConfirm: '将备份当前二进制后替换为 {file} 并重启面板服务（约 5 秒中断）。确认执行？',
  },
  toast: {
    updateStarted: '更新已启动，服务重启中…',
    recovered: '服务已恢复，版本 {version}',
    notRecovered: '服务未在 30 秒内恢复，请检查 journalctl -u ypanel',
    applyFailed: '更新失败',
  },
  online: {
    title: '在线更新',
    desc: 'GitHub / Gitee 双源检查，国内自动走 Gitee 下载',
    check: '检查更新',
    latest: '最新版本',
    unreachable: '不可达',
    noAsset: '无对应架构资产',
    devVersion: '开发版',
    upToDate: '已是最新',
    showNotes: '查看更新日志',
    hideNotes: '收起更新日志',
    checkFailed: '检查更新失败',
    upgradeTitle: '在线升级',
    upgradeConfirm: '将从 {source} 下载 v{version} 并校验后替换重启（约 5~10 秒中断，自动备份 ypanel.bak）。确认升级？',
    upgradeTo: '一键升级到 v{version}',
  },
}
