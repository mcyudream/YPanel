// 主机域节点上下文（M55 遗留补齐）：防火墙/FTP/暴力破解防护等页共享同一节点选择，
// API 经 makeNodeIdApi 包装自动附 nodeId（后端 ?nodeId= 路由）；切换由页面整页刷新。
const KEY = 'host_node'

let current = localStorage.getItem(KEY) || ''

export function setHostNode(n: string) {
  current = n || ''
  if (current) {
    localStorage.setItem(KEY, current)
  } else {
    localStorage.removeItem(KEY)
  }
}

export function getHostNode(): string {
  return current
}

export function withNodeId(url: string): string {
  if (!current || current === 'local') {
    return url
  }
  return url + (url.includes('?') ? '&' : '?') + 'nodeId=' + encodeURIComponent(current)
}

export function makeNodeIdApi(api: {
  get: (u: string, o?: any) => Promise<any>
  post: (u: string, o?: any, o2?: any) => Promise<any>
  put: (u: string, o?: any, o2?: any) => Promise<any>
  delete: (u: string, o?: any) => Promise<any>
}) {
  return {
    get: (u: string, o?: any) => api.get(withNodeId(u), o),
    post: (u: string, o?: any, o2?: any) => api.post(withNodeId(u), o, o2),
    put: (u: string, o?: any, o2?: any) => api.put(withNodeId(u), o, o2),
    delete: (u: string, o?: any) => api.delete(withNodeId(u), o),
  }
}
