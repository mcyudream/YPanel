// processes 域词条（进程与服务）
export default {
  title: '进程与服务',
  desc: '进程与 systemd 服务管理（可选节点）',
  local: '本机',
  localSuffix: '（本机）',
  filterPlaceholder: '筛选…',
  loadErrorTip: '{error}（节点离线或 agent 不可达）',
  loadFailed: '加载失败',
  // 标签页
  tabProcesses: '进程 ({n})',
  tabServices: '服务 ({n})',
  // 进程表
  memCol: '内存%',
  userCol: '用户',
  noProcesses: '无匹配进程',
  kill: '结束',
  killTitle: '结束进程',
  killConfirm: '确认强制结束进程 {name} (PID {pid})？',
  killed: '已结束',
  // 服务表
  serviceCol: '服务',
  loadCol: '加载',
  descCol: '描述',
  noServices: '无匹配服务',
  startDone: '已启动 {name}',
  stopDone: '已停止 {name}',
  restartDone: '已重启 {name}',
  opFailed: '操作失败',
}
