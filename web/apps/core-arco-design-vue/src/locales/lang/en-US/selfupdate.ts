// selfupdate domain messages (panel settings / update channel)
export default {
  page: {
    title: 'Panel Settings',
    desc: 'Update channel: place the new version binary into server {dir} (named with the ypanel prefix), then apply here with one click; ypanel.bak is backed up automatically before updating',
    security: 'Security Settings',
    currentVersion: 'Current version:',
  },
  table: {
    file: 'Update File',
    time: 'Uploaded At',
    empty: 'Update channel is empty: scp the new version binary to {dir} on the server, then refresh',
    applying: 'Updating…',
    applyRestart: 'Apply & Restart',
  },
  modal: {
    applyTitle: 'Apply Update',
    applyConfirm: 'The current binary will be backed up, replaced with {file}, and the panel service restarted (about 5 seconds of downtime). Continue?',
  },
  toast: {
    updateStarted: 'Update started, service restarting…',
    recovered: 'Service recovered, version {version}',
    notRecovered: 'Service did not recover within 30 seconds; check journalctl -u ypanel',
    applyFailed: 'Update failed',
  },
  online: {
    title: 'Online Update',
    desc: 'Checks GitHub / Gitee; downloads from Gitee automatically in mainland China',
    check: 'Check for Updates',
    latest: 'Latest',
    unreachable: 'Unreachable',
    noAsset: 'No asset for this arch',
    devVersion: 'Dev build',
    upToDate: 'Up to date',
    showNotes: 'Show release notes',
    hideNotes: 'Hide release notes',
    checkFailed: 'Failed to check for updates',
    upgradeTitle: 'Online Upgrade',
    upgradeConfirm: 'Download v{version} from {source}, verify, replace and restart (about 5-10 seconds of downtime; ypanel.bak is kept). Continue?',
    upgradeTo: 'Upgrade to v{version}',
  },
}
