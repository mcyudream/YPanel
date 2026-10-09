import api from '../index'

export interface FileEntry {
  name: string
  path: string
  isDir: boolean
  size: number
  mode: string
  modeOct: string
  owner: string
  group: string
  modTime: string
  target?: string
}

export interface FileListResp {
  path: string
  entries: FileEntry[]
}

export interface FileReadResp {
  path: string
  content: string
  size: number
  truncated: boolean
  /** raw=1 时返回：原始字节 base64 */
  contentB64?: string
  /** 实际应用的服务端编码 */
  encoding?: string
  /** 检测为二进制文件 */
  isBinary?: boolean
}

export interface FileOwnersResp {
  users: Array<{ name: string, id: string }>
  groups: Array<{ name: string, id: string }>
}

export interface FileWriteReq {
  path: string
  content: string
  /** 落盘编码，默认 utf-8 */
  encoding?: string
}

// node 为节点 id（默认 local，core 端 ?node= 路由）
function nodeQ(node?: string) {
  return node && node !== 'local' ? `&node=${encodeURIComponent(node)}` : ''
}

export default {
  list: async (path: string, node?: string) => {
    const res = await api.get(`api/v1/files/list?path=${encodeURIComponent(path)}${nodeQ(node)}`)
    return res.data as FileListResp
  },
  read: async (path: string, node?: string, opts?: { raw?: boolean, encoding?: string }) => {
    let q = `api/v1/files/read?path=${encodeURIComponent(path)}${nodeQ(node)}`
    if (opts?.raw) {
      q += '&raw=1'
    }
    if (opts?.encoding) {
      q += `&encoding=${encodeURIComponent(opts.encoding)}`
    }
    const res = await api.get(q)
    return res.data as FileReadResp
  },
  write: (path: string, content: string, node?: string, encoding?: string) => {
    const req: FileWriteReq = { path, content }
    if (encoding) {
      req.encoding = encoding
    }
    return api.post(`api/v1/files/write${nodeQ2(node)}`, req)
  },
  mkdir: (path: string, node?: string) => api.post(`api/v1/files/mkdir${nodeQ2(node)}`, { path }),
  rename: (from: string, to: string, node?: string) => api.post(`api/v1/files/rename${nodeQ2(node)}`, { from, to }),
  copy: (from: string, to: string, node?: string, overwrite?: boolean) =>
    api.post(`api/v1/files/copy${nodeQ2(node)}`, { from, to, overwrite: !!overwrite }),
  delete: (paths: string[], node?: string) => api.post(`api/v1/files/delete${nodeQ2(node)}`, { paths }),
  chmod: (path: string, mode: string, node?: string, recursive?: boolean) =>
    api.post(`api/v1/files/chmod${nodeQ2(node)}`, { path, mode, recursive: !!recursive }),
  chown: (path: string, owner: string, group: string, node?: string, recursive?: boolean) =>
    api.post(`api/v1/files/chown${nodeQ2(node)}`, { path, owner, group, recursive: !!recursive }),
  owners: async (node?: string) => {
    const res = await api.get(`api/v1/files/owners?1=1${nodeQ(node)}`)
    return res.data as FileOwnersResp
  },
  compress: (srcs: string[], dest: string, node?: string) =>
    api.post(`api/v1/files/compress${nodeQ2(node)}`, { srcs, dest }, { timeout: 600000 }),
  decompress: (archive: string, destDir: string, node?: string) => api.post(`api/v1/files/decompress${nodeQ2(node)}`, { archive, destDir }, { timeout: 600000 }),
  search: async (dir: string, keyword: string, node?: string) => {
    const res = await api.get(`api/v1/files/search?dir=${encodeURIComponent(dir)}&keyword=${encodeURIComponent(keyword)}${nodeQ(node)}`, { timeout: 120000 })
    return res.data as FileEntry[]
  },
  upload: (path: string, file: File, onProgress?: (percent: number) => void, node?: string) => {
    const form = new FormData()
    form.append('file', file)
    return api.post(`api/v1/files/upload?path=${encodeURIComponent(path)}${nodeQ(node)}`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          onProgress(Math.round((e.loaded / e.total) * 100))
        }
      },
    })
  },
  // 下载地址（浏览器直开，token 经 query 传递）
  downloadURL: (path: string, token: string, node?: string) =>
    `api/v1/files/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}${nodeQ(node)}`,
}

// POST 类接口的 node 查询串（?node=xxx 或空串）
function nodeQ2(node?: string) {
  const q = nodeQ(node)
  return q ? `?${q.slice(1)}` : ''
}

// ---- M38：回收站 / 收藏 / 分享 / 远程下载 ----
export interface TrashItem {
  original: string
  name: string
  isDir: boolean
  size: number
  trashedAt: string
}

export interface FileFavorite {
  id: number
  path: string
  name: string
  createdAt: string
}

export interface FileShareRow {
  id: number
  path: string
  name: string
  expireAt: string | null
  enabled: boolean
  valid: boolean
}

export const fileExtApi = {
  trash: (paths: string[]) => api.post('api/v1/files/trash', { paths }),
  trashList: async () => {
    const res = await api.get('api/v1/files/trash/list', { silent: true })
    return res.data as TrashItem[]
  },
  trashRestore: (names: string[]) => api.post('api/v1/files/trash/restore', { names }),
  trashPurge: (names: string[]) => api.post('api/v1/files/trash/purge', { names }),
  trashClear: async () => {
    const res = await api.post('api/v1/files/trash/clear')
    return res.data as { count: number }
  },
  favorites: async () => {
    const res = await api.get('api/v1/files/favorites', { silent: true })
    return res.data as FileFavorite[]
  },
  addFavorite: (path: string) => api.post('api/v1/files/favorites', { path }),
  removeFavorite: (id: number) => api.delete(`api/v1/files/favorites/${id}`),
  shares: async () => {
    const res = await api.get('api/v1/files/shares', { silent: true })
    return res.data as FileShareRow[]
  },
  createShare: async (path: string, days: number) => {
    const res = await api.post('api/v1/files/shares', { path, days })
    return res.data as { id: number, token: string, name: string }
  },
  revokeShare: (id: number) => api.delete(`api/v1/files/shares/${id}`),
  remoteDownload: async (url: string, destDir: string) => {
    const res = await api.post('api/v1/files/remote-download', { url, destDir })
    return res.data as { file: string, dir: string }
  },
}
