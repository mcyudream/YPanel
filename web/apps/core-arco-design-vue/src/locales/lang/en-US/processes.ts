// processes domain messages (processes and services)
export default {
  title: 'Processes & Services',
  desc: 'Process and systemd service management (node selectable)',
  local: 'Local',
  localSuffix: ' (local)',
  filterPlaceholder: 'Filter…',
  loadErrorTip: '{error} (node offline or agent unreachable)',
  loadFailed: 'Load failed',
  // tabs
  tabProcesses: 'Processes ({n})',
  tabServices: 'Services ({n})',
  // process table
  memCol: 'MEM%',
  userCol: 'User',
  noProcesses: 'No matching processes',
  kill: 'Kill',
  killTitle: 'Kill Process',
  killConfirm: 'Force kill process {name} (PID {pid})?',
  killed: 'Killed',
  // service table
  serviceCol: 'Service',
  loadCol: 'Load',
  descCol: 'Description',
  noServices: 'No matching services',
  startDone: 'Started {name}',
  stopDone: 'Stopped {name}',
  restartDone: 'Restarted {name}',
  opFailed: 'Operation failed',
}
