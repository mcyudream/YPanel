// 容器域节点上下文（M55）：模块级当前节点 + URL 包装。
// 六个容器 Tab 页共享同一选择状态（localStorage 持久化）；切换后由页面自行刷新数据。
const KEY = 'docker_node'

let current = localStorage.getItem(KEY) || ''

export function setDockerNode(n: string) {
  current = n || ''
  if (current) {
    localStorage.setItem(KEY, current)
  } else {
    localStorage.removeItem(KEY)
  }
}

export function getDockerNode(): string {
  return current
}

// withNode 给 API URL 追加当前节点（空/local 不加，保持向后兼容）
export function withNode(url: string): string {
  if (!current || current === 'local') {
    return url
  }
  return url + (url.includes('?') ? '&' : '?') + 'node=' + encodeURIComponent(current)
}

// napi：带节点路由的 api 包装（容器域专用——get/post/put/delete 自动附 node）
export function makeNodeApi(api: {
  get: (u: string, o?: any) => Promise<any>
  post: (u: string, o?: any, o2?: any) => Promise<any>
  put: (u: string, o?: any, o2?: any) => Promise<any>
  delete: (u: string, o?: any) => Promise<any>
}) {
  return {
    get: (u: string, o?: any) => api.get(withNode(u), o),
    post: (u: string, o?: any, o2?: any) => api.post(withNode(u), o, o2),
    put: (u: string, o?: any, o2?: any) => api.put(withNode(u), o, o2),
    delete: (u: string, o?: any) => api.delete(withNode(u), o),
  }
}
