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
}

export default {
  list: async (path: string) => {
    const res = await api.get(`api/v1/files/list?path=${encodeURIComponent(path)}`)
    return res.data as FileListResp
  },
  read: async (path: string) => {
    const res = await api.get(`api/v1/files/read?path=${encodeURIComponent(path)}`)
    return res.data as FileReadResp
  },
  write: (path: string, content: string) => api.post('api/v1/files/write', { path, content }),
  mkdir: (path: string) => api.post('api/v1/files/mkdir', { path }),
  rename: (from: string, to: string) => api.post('api/v1/files/rename', { from, to }),
  delete: (paths: string[]) => api.post('api/v1/files/delete', { paths }),
  chmod: (path: string, mode: string) => api.post('api/v1/files/chmod', { path, mode }),
  compress: (src: string, dest: string) => api.post('api/v1/files/compress', { src, dest }, { timeout: 600000 }),
  decompress: (archive: string, destDir: string) => api.post('api/v1/files/decompress', { archive, destDir }, { timeout: 600000 }),
  search: async (dir: string, keyword: string) => {
    const res = await api.get(`api/v1/files/search?dir=${encodeURIComponent(dir)}&keyword=${encodeURIComponent(keyword)}`, { timeout: 120000 })
    return res.data as FileEntry[]
  },
  upload: (path: string, file: File, onProgress?: (percent: number) => void) => {
    const form = new FormData()
    form.append('file', file)
    return api.post(`api/v1/files/upload?path=${encodeURIComponent(path)}`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          onProgress(Math.round((e.loaded / e.total) * 100))
        }
      },
    })
  },
  // 下载地址（浏览器直开，token 经 query 传递）
  downloadURL: (path: string, token: string) =>
    `api/v1/files/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}`,
}
