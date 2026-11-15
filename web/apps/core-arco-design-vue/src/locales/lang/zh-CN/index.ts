import ai from './ai'
import aiAssistant from './aiAssistant'
import alert from './alert'
import certs from './certs'
// zh-CN 语言包聚合：按域拆分，新增词条先查 common，再入对应域文件
import common from './common'
import components from './components'
import container from './container'
import cron from './cron'
import database from './database'
import dbadmin from './dbadmin'
import desktop from './desktop'
import diskguard from './diskguard'
import dns from './dns'
import env from './env'
import files from './files'
import firewall from './firewall'
import hosts from './hosts'
import icons from './icons'
import layout from './layout'
import logcenter from './logcenter'
import login from './login'
import manage from './manage'
import menu from './menu'
import monitor from './monitor'
import nat from './nat'
import nodes from './nodes'
import overview from './overview'
import processes from './processes'
import rbac from './rbac'
import runtimes from './runtimes'
import selfupdate from './selfupdate'
import sites from './sites'
import storage from './storage'
import store from './store'
import system from './system'
import terminal from './terminal'
import tools from './tools'
import vpn from './vpn'

export default {
  common,
  menu,
  layout,
  login,
  storage,
  system,
  env,
  cron,
  alert,
  files,
  nodes,
  desktop,
  overview,
  container,
  sites,
  database,
  dbadmin,
  rbac,
  manage,
  monitor,
  logcenter,
  firewall,
  tools,
  nat,
  hosts,
  processes,
  dns,
  vpn,
  terminal,
  certs,
  runtimes,
  store,
  selfupdate,
  diskguard,
  icons,
  ai,
  aiAssistant,
  components,
}
